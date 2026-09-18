package watch

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/LegacyCodeHQ/clarity/cmd/watch/protocol"
	"github.com/LegacyCodeHQ/clarity/cmd/watch/store"
)

// TestBroker_Restart_MatchingState_ResumesOrphanedSession simulates a real
// crash-and-relaunch: one broker publishes a snapshot and is discarded
// without ever closing its session (as if the process died), then a second,
// independent broker attaches to the same db and worktree and publishes the
// identical graph — the state a fresh rebuild on attach would produce if
// nothing changed while the process was down.
func TestBroker_Restart_MatchingState_ResumesOrphanedSession(t *testing.T) {
	db := openPersistenceTestDB(t)

	orphanSessionID := crashMidSession(t, db, "digraph{a}")
	var origRunID int64
	require.NoError(t, db.QueryRow(`SELECT run_id FROM sessions WHERE id = ?`, orphanSessionID).Scan(&origRunID))

	b2, _ := attachBroker(t, db)
	require.NotEqual(t, origRunID, b2.runID, "the resuming process must have opened its own, distinct run")
	b2.publish("main", "digraph{a}")

	var sessionCount int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM sessions WHERE worktree_id = 'main'`).Scan(&sessionCount))
	assert.Equal(t, 1, sessionCount, "a matching restart must resume, not create a second session")

	var closedAt sql.NullTime
	var runID int64
	require.NoError(t, db.QueryRow(`SELECT closed_at, run_id FROM sessions WHERE id = ?`, orphanSessionID).Scan(&closedAt, &runID))
	assert.False(t, closedAt.Valid)
	assert.Equal(t, origRunID, runID, "a resumed session keeps the run_id of the run that opened it, not the run that resumed it")

	var snapshotCount int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM snapshots WHERE session_id = ?`, orphanSessionID).Scan(&snapshotCount))
	assert.Equal(t, 1, snapshotCount, "an identical rebuild on attach must not write a duplicate snapshot")
}

// TestBroker_Restart_DivergedState_ClosesStaleAndStartsFresh mirrors the
// above, but the second broker's rebuilt graph differs from what the
// crashed process last recorded — proving divergence is detected and the
// orphan is closed as stale rather than silently resumed or left open
// forever.
func TestBroker_Restart_DivergedState_ClosesStaleAndStartsFresh(t *testing.T) {
	db := openPersistenceTestDB(t)

	orphanSessionID := crashMidSession(t, db, "digraph{a}")

	b2, _ := attachBroker(t, db)
	b2.publish("main", "digraph{a;b}") // something changed while the process was down

	var sessionCount int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM sessions WHERE worktree_id = 'main'`).Scan(&sessionCount))
	assert.Equal(t, 2, sessionCount)

	var orphanClosedAt sql.NullTime
	var orphanReason string
	require.NoError(t, db.QueryRow(
		`SELECT closed_at, closed_reason FROM sessions WHERE id = ?`, orphanSessionID,
	).Scan(&orphanClosedAt, &orphanReason))
	assert.True(t, orphanClosedAt.Valid)
	assert.Equal(t, "stale", orphanReason)

	var newSessionNumber int
	require.NoError(t, db.QueryRow(`SELECT number FROM sessions WHERE worktree_id = 'main' AND id != ?`, orphanSessionID).Scan(&newSessionNumber))
	assert.Equal(t, 2, newSessionNumber)
}

// crashMidSession simulates a process that published one snapshot and was
// then discarded without closing its session — no defer, no cleanup, just
// like a crash. Returns the orphaned session's id.
func crashMidSession(t *testing.T, db *sql.DB, dot string) int64 {
	t.Helper()
	b, _ := attachBroker(t, db)
	b.publish("main", dot)

	var sessionID int64
	require.NoError(t, db.QueryRow(`SELECT id FROM sessions WHERE worktree_id = 'main'`).Scan(&sessionID))
	return sessionID
}

// attachBroker creates a fresh broker with persistence enabled against db
// and registers the "main" worktree on it, as clarity watch does on
// startup. Each call is independent — nothing carries over between them
// except what's in the database, exactly like separate process runs.
func attachBroker(t *testing.T, db *sql.DB) (*broker, string) {
	t.Helper()
	projectID, err := store.EnsureProject(db, "origin-a")
	require.NoError(t, err)
	// Each call opens its own watch_runs row, exactly as two independent
	// process launches would — nothing here is shared with a prior call.
	runID, err := store.OpenRun(db, projectID, 1)
	require.NoError(t, err)
	b := newBroker()
	b.enablePersistence(db, projectID, runID)
	b.registerWorktree(protocol.WorktreeDescriptor{ID: "main", Path: "/repo", Kind: protocol.WorktreeKindMain, Active: true})
	return b, projectID
}
