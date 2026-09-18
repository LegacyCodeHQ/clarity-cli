package watch

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/LegacyCodeHQ/clarity/cmd/watch/protocol"
	"github.com/LegacyCodeHQ/clarity/cmd/watch/store"
)

func openPersistenceTestDB(t *testing.T) *sql.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	db, err := store.Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return db
}

func TestBroker_RegisterWorktree_WithoutPersistence_IsUnaffected(t *testing.T) {
	// newBroker() (no store) must keep working exactly as it does for every
	// other broker test in this package — persistence is opt-in.
	b := newBroker()
	b.registerWorktree(protocol.WorktreeDescriptor{ID: "main", Path: "/repo", Kind: protocol.WorktreeKindMain})

	b.mu.Lock()
	defer b.mu.Unlock()
	assert.Len(t, b.worktrees, 1)
}

func TestBroker_RegisterWorktree_WithPersistence_PersistsRow(t *testing.T) {
	db := openPersistenceTestDB(t)
	projectID, err := store.EnsureProject(db, "origin-a")
	require.NoError(t, err)

	runID, err := store.OpenRun(db, projectID, 1)
	require.NoError(t, err)

	b := newBroker()
	b.enablePersistence(db, projectID, runID)
	b.registerWorktree(protocol.WorktreeDescriptor{
		ID:    "main",
		Path:  "/repo",
		Label: "my-branch",
		Kind:  protocol.WorktreeKindMain,
	})

	var gotProjectID, path, label string
	require.NoError(t, db.QueryRow(
		`SELECT project_id, path, last_known_label FROM worktrees WHERE id = 'main'`,
	).Scan(&gotProjectID, &path, &label))
	assert.Equal(t, projectID, gotProjectID)
	assert.Equal(t, "/repo", path)
	assert.Equal(t, "my-branch", label)
}

func TestBroker_MarkWorktreeFinished_WithPersistence_SetsDisposedAt(t *testing.T) {
	db := openPersistenceTestDB(t)
	projectID, err := store.EnsureProject(db, "origin-a")
	require.NoError(t, err)

	runID, err := store.OpenRun(db, projectID, 1)
	require.NoError(t, err)

	b := newBroker()
	b.enablePersistence(db, projectID, runID)
	b.registerWorktree(protocol.WorktreeDescriptor{ID: "main", Path: "/repo", Kind: protocol.WorktreeKindMain, Active: true})

	b.markWorktreeFinished("main")

	var disposedAt sql.NullTime
	require.NoError(t, db.QueryRow(`SELECT disposed_at FROM worktrees WHERE id = 'main'`).Scan(&disposedAt))
	assert.True(t, disposedAt.Valid)
}

func TestBroker_CloseWorktree_WithPersistence_SetsHiddenAt(t *testing.T) {
	db := openPersistenceTestDB(t)
	projectID, err := store.EnsureProject(db, "origin-a")
	require.NoError(t, err)

	runID, err := store.OpenRun(db, projectID, 1)
	require.NoError(t, err)

	b := newBroker()
	b.enablePersistence(db, projectID, runID)
	b.registerWorktree(protocol.WorktreeDescriptor{ID: "main", Path: "/repo", Kind: protocol.WorktreeKindMain, Active: true})
	b.markWorktreeFinished("main")

	outcome := b.closeWorktree("main")

	require.Equal(t, closeOK, outcome)
	var hiddenAt sql.NullTime
	require.NoError(t, db.QueryRow(`SELECT hidden_at FROM worktrees WHERE id = 'main'`).Scan(&hiddenAt))
	assert.True(t, hiddenAt.Valid)

	// In-memory behavior must be unchanged by persistence: closeWorktree
	// still removes the worktree from the broker's live state outright.
	b.mu.Lock()
	defer b.mu.Unlock()
	assert.Empty(t, b.worktrees)
}
