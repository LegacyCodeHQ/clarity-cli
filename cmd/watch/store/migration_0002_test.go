package store

import (
	"database/sql"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMigration0002_PreservesExistingRowsAndAddsWorktreeRemoved simulates a
// database that was migrated to v1 (the two-value closed_reason constraint)
// before v2 existed, with real rows already in it, then upgrades it to v2 —
// proving the table rebuild in 0002 doesn't lose or corrupt existing data,
// and that the new value actually becomes usable afterward.
func TestMigration0002_PreservesExistingRowsAndAddsWorktreeRemoved(t *testing.T) {
	db := openTestDB(t)

	m := newMigrator(t, db)
	require.NoError(t, m.Migrate(1))

	_, err := db.Exec(`INSERT INTO projects (id, repo_origin, created_at) VALUES ('p1', 'origin-a', '2026-01-01T00:00:00Z')`)
	require.NoError(t, err)
	_, err = db.Exec(`
		INSERT INTO worktrees (id, project_id, path, kind, first_seen_at)
		VALUES ('main', 'p1', '/repo', 'main', '2026-01-01T00:00:00Z')`)
	require.NoError(t, err)
	_, err = db.Exec(`
		INSERT INTO sessions (worktree_id, number, created_at, closed_at, closed_reason)
		VALUES ('main', 1, '2026-01-01T00:00:00Z', '2026-01-01T01:00:00Z', 'abandoned')`)
	require.NoError(t, err)

	// A value that only exists as of v2 must be rejected under v1.
	_, err = db.Exec(`
		INSERT INTO sessions (worktree_id, number, created_at, closed_at, closed_reason)
		VALUES ('main', 2, '2026-01-01T00:00:00Z', '2026-01-01T01:00:00Z', 'worktree_removed')`)
	require.Error(t, err, "worktree_removed must not be accepted before migration 0002 runs")

	require.NoError(t, m.Migrate(2))

	var closedReason string
	require.NoError(t, db.QueryRow(`SELECT closed_reason FROM sessions WHERE number = 1`).Scan(&closedReason))
	assert.Equal(t, "abandoned", closedReason, "the pre-existing row must survive the rebuild unchanged")

	_, err = db.Exec(`
		INSERT INTO sessions (worktree_id, number, created_at, closed_at, closed_reason)
		VALUES ('main', 2, '2026-01-01T00:00:00Z', '2026-01-01T01:00:00Z', 'worktree_removed')`)
	assert.NoError(t, err, "worktree_removed must be accepted after migration 0002 runs")

	assert.NoError(t, checkForeignKeys(db))
}

// newMigrator wires up the same driver stack migrateFromSource uses, but
// hands back the *migrate.Migrate itself so the test can step through
// individual versions with Migrate(n) instead of jumping straight to the
// latest with Up(). Mirrors production's FK-off-before-any-transaction
// handling (see migrate.go) since this exercises the same 0002 rebuild.
func newMigrator(t *testing.T, db *sql.DB) *migrate.Migrate {
	t.Helper()
	_, err := db.Exec(`PRAGMA foreign_keys = OFF`)
	require.NoError(t, err)

	sourceDriver, err := iofs.New(migrationsFS, "migrations")
	require.NoError(t, err)
	dbDriver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	require.NoError(t, err)
	m, err := migrate.NewWithInstance("iofs", sourceDriver, "sqlite3", dbDriver)
	require.NoError(t, err)
	return m
}
