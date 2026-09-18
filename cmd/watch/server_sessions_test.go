package watch

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/LegacyCodeHQ/clarity/cmd/watch/protocol"
	"github.com/LegacyCodeHQ/clarity/vcs"
)

func TestListSessionsEndpoint_WithoutPersistence_Returns404(t *testing.T) {
	b := newBroker()
	b.registerWorktree(protocol.WorktreeDescriptor{ID: "main", Path: "/repo", Kind: protocol.WorktreeKindMain, Active: true})

	srv := newServer(b, 0, "/repo")
	rec := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/sessions", nil))

	assert.Equal(t, http.StatusNotFound, rec.Code, "no database means no history to show")
}

func TestListSessionsEndpoint_ReturnsSummariesAcrossWorktrees(t *testing.T) {
	b, _ := newPersistedBroker(t)
	b.registerWorktree(protocol.WorktreeDescriptor{ID: "wt-a", Path: "/tmp/wt", Kind: protocol.WorktreeKindLinked, Active: true})
	b.publish("main", "digraph{a}")
	b.publish("wt-a", "digraph{b}")

	srv := newServer(b, 0, "/repo")
	rec := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/sessions", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var got []protocol.PersistedSessionSummary
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Len(t, got, 2, "listing must cover every worktree in the project, not just the one the request happened to be about")

	var worktreeIDs []string
	for _, s := range got {
		worktreeIDs = append(worktreeIDs, s.WorktreeID)
		assert.NotZero(t, s.RunID)
		assert.Equal(t, 1, s.SnapshotCount)
	}
	assert.ElementsMatch(t, []string{"main", "wt-a"}, worktreeIDs)
}

func TestGetSessionEndpoint_ReturnsSnapshotsAndCommits(t *testing.T) {
	b, db := newPersistedBroker(t)
	b.publish("main", "digraph{a}")
	b.archiveWorkingSetWithCommitHistory("main", []vcs.CommitSummary{{Hash: "aaa111", Subject: "first commit"}})

	var sessionID int64
	require.NoError(t, db.QueryRow(`SELECT id FROM sessions WHERE worktree_id = 'main'`).Scan(&sessionID))

	srv := newServer(b, 0, "/repo")
	rec := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, sessionPath(sessionID), nil))
	require.Equal(t, http.StatusOK, rec.Code)

	var got protocol.PersistedSessionDetail
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, sessionID, got.ID)
	require.Len(t, got.Snapshots, 1)
	assert.Equal(t, "digraph{a}", got.Snapshots[0].Source)
	require.Len(t, got.Commits, 1)
	assert.Equal(t, "aaa111", got.Commits[0].Hash)
	assert.Equal(t, "committed", got.ClosedReason)
}

func TestGetSessionEndpoint_UnknownSession_Returns404(t *testing.T) {
	b, _ := newPersistedBroker(t)

	srv := newServer(b, 0, "/repo")
	rec := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/sessions/99999", nil))

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestGetSessionEndpoint_InvalidID_Returns400(t *testing.T) {
	b, _ := newPersistedBroker(t)

	srv := newServer(b, 0, "/repo")
	rec := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/sessions/not-a-number", nil))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func sessionPath(sessionID int64) string {
	return fmt.Sprintf("/sessions/%d", sessionID)
}
