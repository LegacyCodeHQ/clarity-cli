package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMigration0003_RenamesAbandonedToDiscardedAndAddsStale simulates a
// database that was migrated to v2 (where closed_reason still used
// 'abandoned') with a real row already in it, then upgrades it to v3 —
// proving existing 'abandoned' rows are renamed to 'discarded' rather than
// left behind or rejected, and that 'stale' becomes usable afterward.
func TestMigration0003_RenamesAbandonedToDiscardedAndAddsStale(t *testing.T) {
	db := openTestDB(t)

	m := newMigrator(t, db)
	require.NoError(t, m.Migrate(2))

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

	// A value that only exists as of v3 must be rejected under v2.
	_, err = db.Exec(`
		INSERT INTO sessions (worktree_id, number, created_at, closed_at, closed_reason)
		VALUES ('main', 2, '2026-01-01T00:00:00Z', '2026-01-01T01:00:00Z', 'stale')`)
	require.Error(t, err, "stale must not be accepted before migration 0003 runs")

	require.NoError(t, m.Migrate(3))

	var closedReason string
	require.NoError(t, db.QueryRow(`SELECT closed_reason FROM sessions WHERE number = 1`).Scan(&closedReason))
	assert.Equal(t, "discarded", closedReason, "a pre-existing 'abandoned' row must be renamed to 'discarded', not left behind")

	_, err = db.Exec(`
		INSERT INTO sessions (worktree_id, number, created_at, closed_at, closed_reason)
		VALUES ('main', 2, '2026-01-01T00:00:00Z', '2026-01-01T01:00:00Z', 'stale')`)
	assert.NoError(t, err, "stale must be accepted after migration 0003 runs")

	assert.NoError(t, checkForeignKeys(db))
}
