package store

import (
	"database/sql"
	"embed"
	"path/filepath"
	"testing"

	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/broken_migrations/*.sql
var brokenMigrationsFS embed.FS

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	db, err := sql.Open("sqlite3", path+"?_foreign_keys=on")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	return db
}

// openMigratedTestDB is openTestDB plus Migrate, for tests that only care
// about querying/writing against the real schema, not migration behavior
// itself.
func openMigratedTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db := openTestDB(t)
	require.NoError(t, Migrate(db))
	return db
}

func tableNames(t *testing.T, db *sql.DB) []string {
	t.Helper()
	rows, err := db.Query(`SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%'`)
	require.NoError(t, err)
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		require.NoError(t, rows.Scan(&name))
		names = append(names, name)
	}
	require.NoError(t, rows.Err())
	return names
}

func columnNames(t *testing.T, db *sql.DB, table string) []string {
	t.Helper()
	rows, err := db.Query(`SELECT name FROM pragma_table_info(?)`, table)
	require.NoError(t, err)
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		require.NoError(t, rows.Scan(&name))
		names = append(names, name)
	}
	require.NoError(t, rows.Err())
	return names
}

func TestMigrate_FreshDatabase_CreatesExpectedSchema(t *testing.T) {
	db := openTestDB(t)

	require.NoError(t, Migrate(db))

	wantTables := []string{"projects", "worktrees", "sessions", "commits", "snapshots", "watch_runs", "schema_migrations"}
	assert.ElementsMatch(t, wantTables, tableNames(t, db),
		"migration must produce exactly the tables in the ER diagram, plus golang-migrate's own schema_migrations")

	assert.ElementsMatch(t, []string{"id", "repo_origin", "created_at"}, columnNames(t, db, "projects"))
	wantWorktreeColumns := []string{"id", "project_id", "path", "kind", "last_known_label", "first_seen_at", "disposed_at", "hidden_at"}
	assert.ElementsMatch(t, wantWorktreeColumns, columnNames(t, db, "worktrees"))
	wantSessionColumns := []string{"id", "worktree_id", "run_id", "number", "created_at", "closed_at", "closed_reason"}
	assert.ElementsMatch(t, wantSessionColumns, columnNames(t, db, "sessions"))
	assert.ElementsMatch(t, []string{"id", "session_id", "position", "hash", "subject"}, columnNames(t, db, "commits"))
	wantSnapshotColumns := []string{"id", "session_id", "position", "source", "format", "kind", "created_at"}
	assert.ElementsMatch(t, wantSnapshotColumns, columnNames(t, db, "snapshots"))
	wantWatchRunColumns := []string{"id", "project_id", "pid", "started_at", "ended_at"}
	assert.ElementsMatch(t, wantWatchRunColumns, columnNames(t, db, "watch_runs"))
}

func TestMigrate_ReapplyIsSafeNoOp(t *testing.T) {
	db := openTestDB(t)

	require.NoError(t, Migrate(db))
	before := tableNames(t, db)

	require.NoError(t, Migrate(db), "re-running Migrate against an already-migrated database must not error")

	assert.ElementsMatch(t, before, tableNames(t, db), "schema must be unchanged by the no-op re-run")
}

func TestMigrate_InsertedRowSurvivesReapply(t *testing.T) {
	db := openTestDB(t)
	require.NoError(t, Migrate(db))

	_, err := db.Exec(`INSERT INTO projects (id, repo_origin, created_at) VALUES ('p1', 'origin1', '2026-01-01T00:00:00Z')`)
	require.NoError(t, err)

	require.NoError(t, Migrate(db))

	var count int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM projects`).Scan(&count))
	assert.Equal(t, 1, count, "re-running migrations must not touch existing data")
}

func TestMigrate_FailurePartwayLeavesCleanState(t *testing.T) {
	db := openTestDB(t)

	brokenSource, err := iofs.New(brokenMigrationsFS, "testdata/broken_migrations")
	require.NoError(t, err)

	err = migrateFromSource(db, brokenSource)
	require.Error(t, err, "a migration with invalid SQL must surface as an error, never be silently swallowed")

	// Migration 0001 (valid) should have committed; migration 0002's failed
	// statement (after its own valid CREATE TABLE bar) must not have left a
	// partial "bar" table behind — golang-migrate wraps each migration file
	// in its own transaction for the sqlite3 driver.
	names := tableNames(t, db)
	assert.Contains(t, names, "foo", "the migration before the broken one must still be applied")
	assert.NotContains(t, names, "bar", "the broken migration's own transaction must have rolled back, not left a partial table")

	// The database must be left in a state that's honest about needing
	// attention, not one that silently claims to be up to date.
	var dirty bool
	var version int
	require.NoError(t, db.QueryRow(`SELECT version, dirty FROM schema_migrations`).Scan(&version, &dirty))
	assert.True(t, dirty, "schema_migrations must record the dirty state left by the failed migration")

	// Retrying the same broken source must keep failing loudly rather than
	// pretend the database is usable.
	err = migrateFromSource(db, brokenSource)
	assert.Error(t, err, "retrying against a dirty database must not silently succeed")
}
