package store

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/LegacyCodeHQ/clarity/cmd/watch/protocol"
)

func TestListSessions_ReturnsEverySessionAcrossWorktreesInProject(t *testing.T) {
	db := openMigratedTestDB(t)
	seedWorktree(t, db, "main")
	runID := seedRun(t, db)
	seedWorktree(t, db, "wt-a")

	mainID, err := OpenSession(db, "main", runID)
	require.NoError(t, err)
	otherID, err := OpenSession(db, "wt-a", runID)
	require.NoError(t, err)

	projectID := seedProject(t, db) // idempotent: same project both worktrees joined
	summaries, err := ListSessions(db, projectID)
	require.NoError(t, err)

	require.Len(t, summaries, 2)
	ids := []int64{summaries[0].ID, summaries[1].ID}
	assert.ElementsMatch(t, []int64{mainID, otherID}, ids, "listing must include sessions from every worktree in the project, not just one")
	for _, s := range summaries {
		assert.Equal(t, runID, s.RunID)
		assert.Nil(t, s.ClosedAt, "an open session must report no close time")
		assert.Empty(t, s.ClosedReason)
	}
}

func TestListSessions_ExcludesSessionsFromOtherProjects(t *testing.T) {
	db := openMigratedTestDB(t)
	seedWorktree(t, db, "main")
	runID := seedRun(t, db)
	projectID := seedProject(t, db)
	_, err := OpenSession(db, "main", runID)
	require.NoError(t, err)

	otherProjectID, err := EnsureProject(db, "origin-b")
	require.NoError(t, err)
	require.NoError(t, RegisterWorktree(db, "other-main", otherProjectID, "/other", protocol.WorktreeKindMain, ""))
	otherRunID, err := OpenRun(db, otherProjectID, 2)
	require.NoError(t, err)
	_, err = OpenSession(db, "other-main", otherRunID)
	require.NoError(t, err)

	summaries, err := ListSessions(db, projectID)
	require.NoError(t, err)

	require.Len(t, summaries, 1, "a project's listing must not include another project's sessions")
	assert.Equal(t, "main", summaries[0].WorktreeID)
}

func TestListSessions_ReportsSnapshotAndCommitCounts(t *testing.T) {
	db := openMigratedTestDB(t)
	seedWorktree(t, db, "main")
	runID := seedRun(t, db)
	projectID := seedProject(t, db)

	sessionID, err := OpenSession(db, "main", runID)
	require.NoError(t, err)
	require.NoError(t, AppendSnapshot(db, sessionID, 0, "digraph{a}", "dot", "baseline", time.Now().UTC()))
	require.NoError(t, AppendSnapshot(db, sessionID, 1, "digraph{a;b}", "dot", "incremental", time.Now().UTC()))
	require.NoError(t, CloseSessionCommitted(db, sessionID, []CommitRecord{{Position: 0, Hash: "aaa111", Subject: "first"}}))

	summaries, err := ListSessions(db, projectID)
	require.NoError(t, err)

	require.Len(t, summaries, 1)
	s := summaries[0]
	assert.Equal(t, 2, s.SnapshotCount)
	assert.Equal(t, 1, s.CommitCount)
	require.NotNil(t, s.ClosedAt)
	assert.Equal(t, "committed", s.ClosedReason)
}

func TestGetSessionDetail_ReturnsSnapshotsAndCommitsInPositionOrder(t *testing.T) {
	db := openMigratedTestDB(t)
	seedWorktree(t, db, "main")
	runID := seedRun(t, db)

	sessionID, err := OpenSession(db, "main", runID)
	require.NoError(t, err)
	require.NoError(t, AppendSnapshot(db, sessionID, 0, "digraph{a}", "dot", "baseline", time.Now().UTC()))
	require.NoError(t, AppendSnapshot(db, sessionID, 1, "digraph{a;b}", "dot", "incremental", time.Now().UTC()))
	commits := []CommitRecord{
		{Position: 0, Hash: "aaa111", Subject: "first"},
		{Position: 1, Hash: "bbb222", Subject: "second"},
	}
	require.NoError(t, CloseSessionCommitted(db, sessionID, commits))

	detail, found, err := GetSessionDetail(db, sessionID)
	require.NoError(t, err)
	require.True(t, found)

	require.Len(t, detail.Snapshots, 2)
	assert.Equal(t, "digraph{a}", detail.Snapshots[0].Source)
	assert.Equal(t, "baseline", detail.Snapshots[0].Kind)
	assert.Equal(t, "digraph{a;b}", detail.Snapshots[1].Source)
	assert.Equal(t, "incremental", detail.Snapshots[1].Kind)

	require.Len(t, detail.Commits, 2)
	assert.Equal(t, commits, detail.Commits)

	assert.Equal(t, sessionID, detail.ID)
	assert.Equal(t, runID, detail.RunID)
}

func TestGetSessionDetail_UnknownSession_ReturnsNotFound(t *testing.T) {
	db := openMigratedTestDB(t)

	_, found, err := GetSessionDetail(db, 99999)
	require.NoError(t, err)
	assert.False(t, found)
}
