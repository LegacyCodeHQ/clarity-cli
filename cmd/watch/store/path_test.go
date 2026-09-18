package store

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// initRepoWithCommit creates a fresh git repo with one commit, so it has a
// resolvable common dir, and returns its absolute path.
func initRepoWithCommit(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
		{"git", "commit", "--allow-empty", "-m", "init"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "cmd %v failed: %s", args, out)
	}
	return dir
}

func TestPathFor_ResolvesUnderCommonGitDir(t *testing.T) {
	repoPath := initRepoWithCommit(t)

	path, err := PathFor(repoPath)
	require.NoError(t, err)

	// git rev-parse resolves symlinks (e.g. macOS's /var -> /private/var),
	// so resolve repoPath the same way before comparing.
	resolvedRepoPath, err := filepath.EvalSymlinks(repoPath)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(resolvedRepoPath, ".git", "clarity", "session-history.db"), path)

	info, err := os.Stat(filepath.Dir(path))
	require.NoError(t, err, "PathFor must create the containing directory")
	assert.True(t, info.IsDir())
}

func TestLockPathFor_SitsBesideTheDatabase(t *testing.T) {
	repoPath := initRepoWithCommit(t)

	dbPath, err := PathFor(repoPath)
	require.NoError(t, err)
	lockPath, err := LockPathFor(repoPath)
	require.NoError(t, err)

	assert.Equal(t, filepath.Dir(dbPath), filepath.Dir(lockPath), "lock file must live alongside the database")
	assert.Equal(t, "watch.lock", filepath.Base(lockPath))
}

func TestLockPathFor_SharedAcrossLinkedWorktree(t *testing.T) {
	repoPath := initRepoWithCommit(t)
	linkedPath := filepath.Join(t.TempDir(), "linked")

	cmd := exec.Command("git", "worktree", "add", "-b", "feature", linkedPath)
	cmd.Dir = repoPath
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git worktree add failed: %s", out)

	mainLockPath, err := LockPathFor(repoPath)
	require.NoError(t, err)
	linkedLockPath, err := LockPathFor(linkedPath)
	require.NoError(t, err)

	assert.Equal(t, mainLockPath, linkedLockPath, "main and linked worktrees of the same clone must resolve to the same lock file")
}

func TestPathFor_SharedAcrossLinkedWorktree(t *testing.T) {
	repoPath := initRepoWithCommit(t)
	linkedPath := filepath.Join(t.TempDir(), "linked")

	cmd := exec.Command("git", "worktree", "add", "-b", "feature", linkedPath)
	cmd.Dir = repoPath
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git worktree add failed: %s", out)

	mainDBPath, err := PathFor(repoPath)
	require.NoError(t, err)
	linkedDBPath, err := PathFor(linkedPath)
	require.NoError(t, err)

	assert.Equal(t, mainDBPath, linkedDBPath, "main and linked worktrees of the same clone must resolve to the same db file")
}
