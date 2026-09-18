package store

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenOrResumeSession_NoOpenSession_BehavesLikeOpenSession(t *testing.T) {
	db := openMigratedTestDB(t)
	seedWorktree(t, db, "main")

	sessionID, nextPosition, matched, err := OpenOrResumeSession(db, "main", "digraph{a}")
	require.NoError(t, err)

	assert.False(t, matched)
	assert.Zero(t, nextPosition)

	var number int
	var closedAt sql.NullTime
	require.NoError(t, db.QueryRow(`SELECT number, closed_at FROM sessions WHERE id = ?`, sessionID).Scan(&number, &closedAt))
	assert.Equal(t, 1, number)
	assert.False(t, closedAt.Valid)
}

func TestOpenOrResumeSession_OrphanWithMatchingSource_Resumes(t *testing.T) {
	db := openMigratedTestDB(t)
	seedWorktree(t, db, "main")

	// Simulate a prior process run: opened a session, wrote two snapshots,
	// then crashed before ever closing it.
	orphanID, err := OpenSession(db, "main")
	require.NoError(t, err)
	require.NoError(t, AppendSnapshot(db, orphanID, 0, "digraph{a}", "dot", "baseline", time.Now().UTC()))
	require.NoError(t, AppendSnapshot(db, orphanID, 1, "digraph{a;b}", "dot", "incremental", time.Now().UTC()))

	sessionID, nextPosition, matched, err := OpenOrResumeSession(db, "main", "digraph{a;b}")
	require.NoError(t, err)

	assert.True(t, matched, "identical rebuilt graph must resume the orphaned session")
	assert.Equal(t, orphanID, sessionID)
	assert.Equal(t, 2, nextPosition, "next position must continue after the orphan's last one, not restart at 0")

	var closedAt sql.NullTime
	require.NoError(t, db.QueryRow(`SELECT closed_at FROM sessions WHERE id = ?`, orphanID).Scan(&closedAt))
	assert.False(t, closedAt.Valid, "a resumed session must not be closed")

	var sessionCount int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM sessions WHERE worktree_id = 'main'`).Scan(&sessionCount))
	assert.Equal(t, 1, sessionCount, "resuming must not create a second session")
}

func TestOpenOrResumeSession_OrphanWithDifferentSource_ClosesStaleAndOpensNew(t *testing.T) {
	db := openMigratedTestDB(t)
	seedWorktree(t, db, "main")

	orphanID, err := OpenSession(db, "main")
	require.NoError(t, err)
	require.NoError(t, AppendSnapshot(db, orphanID, 0, "digraph{a}", "dot", "baseline", time.Now().UTC()))

	sessionID, nextPosition, matched, err := OpenOrResumeSession(db, "main", "digraph{a;b;c}")
	require.NoError(t, err)

	assert.False(t, matched)
	assert.Zero(t, nextPosition)
	assert.NotEqual(t, orphanID, sessionID, "a mismatched orphan must not be resumed")

	var orphanClosedAt sql.NullTime
	var orphanReason string
	require.NoError(t, db.QueryRow(
		`SELECT closed_at, closed_reason FROM sessions WHERE id = ?`, orphanID,
	).Scan(&orphanClosedAt, &orphanReason))
	assert.True(t, orphanClosedAt.Valid)
	assert.Equal(t, "stale", orphanReason)

	var newNumber int
	require.NoError(t, db.QueryRow(`SELECT number FROM sessions WHERE id = ?`, sessionID).Scan(&newNumber))
	assert.Equal(t, 2, newNumber, "the fresh session must still be numbered after the closed orphan")
}

func TestOpenOrResumeSession_OrphanWithNoSnapshots_ClosesStaleAndOpensNew(t *testing.T) {
	db := openMigratedTestDB(t)
	seedWorktree(t, db, "main")

	// A session that exists but never got a single snapshot written — as
	// if the process crashed between OpenSession and the first
	// AppendSnapshot. Nothing to compare against, so it can't be resumed.
	orphanID, err := OpenSession(db, "main")
	require.NoError(t, err)

	sessionID, _, matched, err := OpenOrResumeSession(db, "main", "digraph{a}")
	require.NoError(t, err)

	assert.False(t, matched)
	assert.NotEqual(t, orphanID, sessionID)

	var orphanReason string
	require.NoError(t, db.QueryRow(`SELECT closed_reason FROM sessions WHERE id = ?`, orphanID).Scan(&orphanReason))
	assert.Equal(t, "stale", orphanReason)
}

func TestOpenOrResumeSession_OnlyLatestOpenSessionIsCandidate(t *testing.T) {
	db := openMigratedTestDB(t)
	seedWorktree(t, db, "main")

	// A pre-existing anomaly: an older open session that isn't the most
	// recent one for this worktree (shouldn't normally happen, but the
	// function must not get confused by it).
	older, err := OpenSession(db, "main")
	require.NoError(t, err)
	require.NoError(t, AppendSnapshot(db, older, 0, "digraph{old}", "dot", "baseline", time.Now().UTC()))

	latest, err := OpenSession(db, "main")
	require.NoError(t, err)
	require.NoError(t, AppendSnapshot(db, latest, 0, "digraph{new}", "dot", "baseline", time.Now().UTC()))

	sessionID, _, matched, err := OpenOrResumeSession(db, "main", "digraph{new}")
	require.NoError(t, err)

	assert.True(t, matched)
	assert.Equal(t, latest, sessionID, "only the highest-numbered open session is a resume candidate")

	// The older, un-considered orphan is left exactly as it was.
	var olderClosedAt sql.NullTime
	require.NoError(t, db.QueryRow(`SELECT closed_at FROM sessions WHERE id = ?`, older).Scan(&olderClosedAt))
	assert.False(t, olderClosedAt.Valid)
}
