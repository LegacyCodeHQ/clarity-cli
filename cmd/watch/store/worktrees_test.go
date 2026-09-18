package store

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/LegacyCodeHQ/clarity/cmd/watch/protocol"
)

func seedProject(t *testing.T, db *sql.DB) string {
	t.Helper()
	id, err := EnsureProject(db, "origin-a")
	require.NoError(t, err)
	return id
}

// seedRun opens a watch_runs row against the shared "origin-a" test
// project (EnsureProject is idempotent, so this shares a row with any
// seedWorktree call against the same db) and returns its id — the minimum
// a test needs to call OpenSession/OpenOrResumeSession directly.
func seedRun(t *testing.T, db *sql.DB) int64 {
	t.Helper()
	projectID := seedProject(t, db)
	runID, err := OpenRun(db, projectID, 1)
	require.NoError(t, err)
	return runID
}

func worktreeRow(t *testing.T, db *sql.DB, id string) (
	projectID, path, kind string,
	lastKnownLabel sql.NullString,
	firstSeenAt time.Time,
	disposedAt, hiddenAt sql.NullTime,
) {
	t.Helper()
	querySQL := `SELECT project_id, path, kind, last_known_label, first_seen_at, disposed_at, hidden_at
		FROM worktrees WHERE id = ?`
	err := db.QueryRow(querySQL, id).Scan(&projectID, &path, &kind, &lastKnownLabel, &firstSeenAt, &disposedAt, &hiddenAt)
	require.NoError(t, err)
	return
}

func TestRegisterWorktree_FirstCall_InsertsRow(t *testing.T) {
	db := openMigratedTestDB(t)
	projectID := seedProject(t, db)

	require.NoError(t, RegisterWorktree(db, "main", projectID, "/repo", protocol.WorktreeKindMain, "my-branch"))

	gotProjectID, path, kind, label, firstSeenAt, disposedAt, hiddenAt := worktreeRow(t, db, "main")
	assert.Equal(t, projectID, gotProjectID)
	assert.Equal(t, "/repo", path)
	assert.Equal(t, "main", kind)
	assert.Equal(t, "my-branch", label.String)
	assert.False(t, firstSeenAt.IsZero())
	assert.False(t, disposedAt.Valid)
	assert.False(t, hiddenAt.Valid)
}

func TestRegisterWorktree_Reregistration_RefreshesLabelPreservesFirstSeenAt(t *testing.T) {
	db := openMigratedTestDB(t)
	projectID := seedProject(t, db)

	require.NoError(t, RegisterWorktree(db, "main", projectID, "/repo", protocol.WorktreeKindMain, "old-branch"))
	_, _, _, _, firstSeenAt1, _, _ := worktreeRow(t, db, "main")

	time.Sleep(10 * time.Millisecond)
	require.NoError(t, RegisterWorktree(db, "main", projectID, "/repo", protocol.WorktreeKindMain, "new-branch"))
	_, _, _, label, firstSeenAt2, _, _ := worktreeRow(t, db, "main")

	assert.Equal(t, "new-branch", label.String, "re-registration must refresh last_known_label")
	assert.Equal(t, firstSeenAt1, firstSeenAt2, "re-registration must not move first_seen_at")

	var count int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM worktrees WHERE id = 'main'`).Scan(&count))
	assert.Equal(t, 1, count, "re-registration must not insert a duplicate row")
}

func TestRegisterWorktree_Reregistration_ClearsDisposedAndHidden(t *testing.T) {
	db := openMigratedTestDB(t)
	projectID := seedProject(t, db)
	require.NoError(t, RegisterWorktree(db, "main", projectID, "/repo", protocol.WorktreeKindMain, "label"))
	require.NoError(t, MarkWorktreeDisposed(db, "main"))
	require.NoError(t, MarkWorktreeHidden(db, "main"))

	require.NoError(t, RegisterWorktree(db, "main", projectID, "/repo", protocol.WorktreeKindMain, "label"))

	_, _, _, _, _, disposedAt, hiddenAt := worktreeRow(t, db, "main")
	assert.False(t, disposedAt.Valid, "a live re-registration contradicts a prior disposed marker")
	assert.False(t, hiddenAt.Valid, "a live re-registration contradicts a prior hidden marker")
}

func TestMarkWorktreeDisposed_SetsDisposedAt_RowSurvives(t *testing.T) {
	db := openMigratedTestDB(t)
	projectID := seedProject(t, db)
	require.NoError(t, RegisterWorktree(db, "main", projectID, "/repo", protocol.WorktreeKindMain, "label"))

	require.NoError(t, MarkWorktreeDisposed(db, "main"))

	_, _, _, _, _, disposedAt, _ := worktreeRow(t, db, "main")
	assert.True(t, disposedAt.Valid)
}

func TestMarkWorktreeHidden_SetsHiddenAt_RowSurvives(t *testing.T) {
	db := openMigratedTestDB(t)
	projectID := seedProject(t, db)
	require.NoError(t, RegisterWorktree(db, "main", projectID, "/repo", protocol.WorktreeKindMain, "label"))
	require.NoError(t, MarkWorktreeDisposed(db, "main"))

	require.NoError(t, MarkWorktreeHidden(db, "main"))

	_, _, _, _, _, _, hiddenAt := worktreeRow(t, db, "main")
	assert.True(t, hiddenAt.Valid)

	var count int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM worktrees WHERE id = 'main'`).Scan(&count))
	assert.Equal(t, 1, count, "hiding a worktree must never delete its row")
}
