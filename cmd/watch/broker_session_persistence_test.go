package watch

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/LegacyCodeHQ/clarity/cmd/watch/protocol"
	"github.com/LegacyCodeHQ/clarity/cmd/watch/store"
	"github.com/LegacyCodeHQ/clarity/vcs"
)

func newPersistedBroker(t *testing.T) (*broker, *sql.DB) {
	t.Helper()
	db := openPersistenceTestDB(t)
	projectID, err := store.EnsureProject(db, "origin-a")
	require.NoError(t, err)

	runID, err := store.OpenRun(db, projectID, 1)
	require.NoError(t, err)

	b := newBroker()
	b.enablePersistence(db, projectID, runID)
	b.registerWorktree(protocol.WorktreeDescriptor{ID: "main", Path: "/repo", Kind: protocol.WorktreeKindMain, Active: true})
	return b, db
}

func TestBroker_Publish_FirstSnapshot_OpensSessionNumberOne(t *testing.T) {
	b, db := newPersistedBroker(t)

	b.publish("main", "digraph{a}")

	var number int
	var closedAt sql.NullTime
	require.NoError(t, db.QueryRow(`SELECT number, closed_at FROM sessions WHERE worktree_id = 'main'`).Scan(&number, &closedAt))
	assert.Equal(t, 1, number)
	assert.False(t, closedAt.Valid, "a session with its first snapshot must still be open")

	var snapshotCount int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM snapshots`).Scan(&snapshotCount))
	assert.Equal(t, 1, snapshotCount)
}

func TestBroker_Publish_EachSnapshot_PersistedIndividually(t *testing.T) {
	b, db := newPersistedBroker(t)

	b.publish("main", "digraph{a}")
	b.publish("main", "digraph{a;b}")
	b.publish("main", "digraph{a;b;c}")

	// Simulates checking durable state mid-stream, as if the process were
	// killed right after the third snapshot with no commit yet — every
	// snapshot published so far must already be queryable, not buffered
	// in memory waiting for a session close.
	rows, err := db.Query(`SELECT position, source, kind FROM snapshots ORDER BY position`)
	require.NoError(t, err)
	defer rows.Close()

	type row struct {
		position int
		source   string
		kind     string
	}
	var got []row
	for rows.Next() {
		var r row
		require.NoError(t, rows.Scan(&r.position, &r.source, &r.kind))
		got = append(got, r)
	}
	require.Len(t, got, 3)
	assert.Equal(t, "baseline", got[0].kind, "the first snapshot of the process's lifetime for this worktree is the baseline")
	assert.Equal(t, "incremental", got[1].kind)
	assert.Equal(t, "incremental", got[2].kind)
	assert.Equal(t, []int{0, 1, 2}, []int{got[0].position, got[1].position, got[2].position})
}

func TestBroker_ArchiveWorkingSetWithCommitHistory_ClosesSessionCommitted(t *testing.T) {
	b, db := newPersistedBroker(t)
	b.publish("main", "digraph{a}")

	b.archiveWorkingSetWithCommitHistory("main", []vcs.CommitSummary{
		{Hash: "aaa111", Subject: "first commit"},
	})

	var closedAt sql.NullTime
	var closedReason string
	require.NoError(t, db.QueryRow(`SELECT closed_at, closed_reason FROM sessions WHERE number = 1`).Scan(&closedAt, &closedReason))
	assert.True(t, closedAt.Valid)
	assert.Equal(t, "committed", closedReason)

	var commitCount int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM commits`).Scan(&commitCount))
	assert.Equal(t, 1, commitCount)
}

// TestBroker_ArchiveWorkingSet_BroadcastCollectionCarriesPersistedSessionID
// proves the in-memory collection the SSE payload broadcasts is
// recognizable as the very session that also lands in the persisted-history
// listing (CLR-98's GET /sessions) — the frontend needs this to avoid
// showing the current run's own closed sessions twice.
func TestBroker_ArchiveWorkingSet_BroadcastCollectionCarriesPersistedSessionID(t *testing.T) {
	b, db := newPersistedBroker(t)
	b.publish("main", "digraph{a}")

	ch := b.subscribe()
	<-ch // drain the initial payload subscribe() seeds, sent before the archive below

	b.archiveWorkingSetWithCommitHistory("main", []vcs.CommitSummary{{Hash: "aaa111", Subject: "first commit"}})
	payload := <-ch

	var wantSessionID int64
	require.NoError(t, db.QueryRow(`SELECT id FROM sessions WHERE number = 1`).Scan(&wantSessionID))

	require.Len(t, payload.PastCollections, 1)
	assert.Equal(t, wantSessionID, payload.PastCollections[0].SessionID)
}

func TestBroker_ClearWorkingSet_ClosesSessionDiscarded(t *testing.T) {
	b, db := newPersistedBroker(t)
	b.publish("main", "digraph{a}")

	b.clearWorkingSet("main")

	var closedAt sql.NullTime
	var closedReason string
	require.NoError(t, db.QueryRow(`SELECT closed_at, closed_reason FROM sessions WHERE number = 1`).Scan(&closedAt, &closedReason))
	assert.True(t, closedAt.Valid)
	assert.Equal(t, "discarded", closedReason)

	var commitCount int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM commits`).Scan(&commitCount))
	assert.Zero(t, commitCount)
}

func TestBroker_MarkWorktreeFinished_WithPendingSnapshots_ClosesSessionWorktreeRemoved(t *testing.T) {
	b, db := newPersistedBroker(t)
	b.publish("main", "digraph{a}")

	b.markWorktreeFinished("main")

	var closedReason string
	require.NoError(t, db.QueryRow(`SELECT closed_reason FROM sessions WHERE number = 1`).Scan(&closedReason))
	assert.Equal(t, "worktree_removed", closedReason)
}

func TestBroker_MarkWorktreeFinished_WithNoPendingSnapshots_ClosesNoSession(t *testing.T) {
	b, db := newPersistedBroker(t)
	b.publish("main", "digraph{a}")
	b.archiveWorkingSetWithCommitHistory("main", []vcs.CommitSummary{{Hash: "aaa111", Subject: "commit"}})

	// Nothing pending after the commit closed the session — markWorktreeFinished
	// must not create or touch any session.
	b.markWorktreeFinished("main")

	var sessionCount int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM sessions`).Scan(&sessionCount))
	assert.Equal(t, 1, sessionCount, "no new session should be opened or closed for an already-clean worktree")
}

func TestBroker_NextSnapshotAfterClose_OpensNewSessionIncrementedNumber(t *testing.T) {
	b, db := newPersistedBroker(t)
	b.publish("main", "digraph{a}")
	b.archiveWorkingSetWithCommitHistory("main", []vcs.CommitSummary{{Hash: "aaa111", Subject: "commit"}})

	b.publish("main", "digraph{a;b}")

	var count int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM sessions`).Scan(&count))
	assert.Equal(t, 2, count)

	var secondNumber int
	var closedAt sql.NullTime
	require.NoError(t, db.QueryRow(`SELECT number, closed_at FROM sessions WHERE number = 2`).Scan(&secondNumber, &closedAt))
	assert.Equal(t, 2, secondNumber)
	assert.False(t, closedAt.Valid)
}
