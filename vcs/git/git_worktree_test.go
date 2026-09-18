package git

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetCommonDir_Main(t *testing.T) {
	repo := t.TempDir()
	setupGitRepo(t, repo)
	seedInitialCommit(t, repo)

	common, err := GetCommonDir(repo)
	require.NoError(t, err)

	want := resolveSymlinks(filepath.Join(repo, ".git"))
	assert.Equal(t, want, resolveSymlinks(common))
}

func TestGetCommonDir_LinkedWorktree(t *testing.T) {
	repo := t.TempDir()
	setupGitRepo(t, repo)
	seedInitialCommit(t, repo)

	wtParent := t.TempDir()
	wt := filepath.Join(wtParent, "linked")
	gitWorktreeAdd(t, repo, wt, "feat/linked")

	commonFromMain, err := GetCommonDir(repo)
	require.NoError(t, err)
	commonFromLinked, err := GetCommonDir(wt)
	require.NoError(t, err)

	assert.Equal(t, resolveSymlinks(commonFromMain), resolveSymlinks(commonFromLinked),
		"common dir must be identical for the main worktree and linked worktrees")
}

func TestGetWorktreeRoot_ResolvesSubdirectoryToRoot(t *testing.T) {
	repo := t.TempDir()
	setupGitRepo(t, repo)
	seedInitialCommit(t, repo)

	sub := filepath.Join(repo, "sub", "deeper")
	require.NoError(t, exec.Command("mkdir", "-p", sub).Run())

	root, err := GetWorktreeRoot(sub)
	require.NoError(t, err)
	assert.Equal(t, resolveSymlinks(repo), resolveSymlinks(root))
}

func TestGetWorktreeRoot_LinkedWorktreeIsItsOwnRoot(t *testing.T) {
	repo := t.TempDir()
	setupGitRepo(t, repo)
	seedInitialCommit(t, repo)

	wtParent := t.TempDir()
	wt := filepath.Join(wtParent, "linked")
	gitWorktreeAdd(t, repo, wt, "feat/linked")

	root, err := GetWorktreeRoot(wt)
	require.NoError(t, err)
	assert.Equal(t, resolveSymlinks(wt), resolveSymlinks(root),
		"a linked worktree's root must be itself, not the main worktree")
}

func TestGetGitDir_MainEqualsCommon(t *testing.T) {
	repo := t.TempDir()
	setupGitRepo(t, repo)
	seedInitialCommit(t, repo)

	gitDir, err := GetGitDir(repo)
	require.NoError(t, err)
	common, err := GetCommonDir(repo)
	require.NoError(t, err)

	assert.Equal(t, resolveSymlinks(common), resolveSymlinks(gitDir))
}

func TestGetGitDir_LinkedDiffersFromCommon(t *testing.T) {
	repo := t.TempDir()
	setupGitRepo(t, repo)
	seedInitialCommit(t, repo)

	wt := filepath.Join(t.TempDir(), "linked")
	gitWorktreeAdd(t, repo, wt, "feat/linked")

	gitDir, err := GetGitDir(wt)
	require.NoError(t, err)
	common, err := GetCommonDir(wt)
	require.NoError(t, err)

	assert.NotEqual(t, resolveSymlinks(common), resolveSymlinks(gitDir),
		"linked worktree gitdir should be the per-worktree dir, not the common one")
	assert.True(t, strings.Contains(resolveSymlinks(gitDir), "worktrees"),
		"linked gitdir should live under <common>/worktrees/<name>; got %q", gitDir)
}

func TestWorktreeKindFor(t *testing.T) {
	repo := t.TempDir()
	setupGitRepo(t, repo)
	seedInitialCommit(t, repo)

	wt := filepath.Join(t.TempDir(), "linked")
	gitWorktreeAdd(t, repo, wt, "feat/linked")

	main, err := WorktreeKindFor(repo)
	require.NoError(t, err)
	assert.Equal(t, WorktreeKindMain, main, "the main worktree must report WorktreeKindMain")

	linked, err := WorktreeKindFor(wt)
	require.NoError(t, err)
	assert.Equal(t, WorktreeKindLinked, linked, "a linked worktree must report WorktreeKindLinked")
}

func TestListWorktrees_MainOnly(t *testing.T) {
	repo := t.TempDir()
	setupGitRepo(t, repo)
	seedInitialCommit(t, repo)

	wts, err := ListWorktrees(repo)
	require.NoError(t, err)
	require.Len(t, wts, 1)

	assert.Equal(t, WorktreeKindMain, wts[0].Kind)
	assert.Equal(t, resolveSymlinks(repo), resolveSymlinks(wts[0].Path))
	assert.NotEmpty(t, wts[0].Head)
}

func TestListWorktrees_MainAndLinked(t *testing.T) {
	repo := t.TempDir()
	setupGitRepo(t, repo)
	seedInitialCommit(t, repo)

	wt := filepath.Join(t.TempDir(), "linked")
	gitWorktreeAdd(t, repo, wt, "feat/linked")

	wts, err := ListWorktrees(repo)
	require.NoError(t, err)
	require.Len(t, wts, 2)

	byPath := make(map[string]Worktree)
	for _, w := range wts {
		byPath[resolveSymlinks(w.Path)] = w
	}

	main, ok := byPath[resolveSymlinks(repo)]
	require.True(t, ok, "the main worktree should appear in list")
	assert.Equal(t, WorktreeKindMain, main.Kind)

	linked, ok := byPath[resolveSymlinks(wt)]
	require.True(t, ok, "linked worktree should appear in list")
	assert.Equal(t, WorktreeKindLinked, linked.Kind)
	assert.Equal(t, "refs/heads/feat/linked", linked.Branch)
}

func TestListWorktrees_DetachedHead(t *testing.T) {
	repo := t.TempDir()
	setupGitRepo(t, repo)
	seedInitialCommit(t, repo)

	wt := filepath.Join(t.TempDir(), "detached")
	// Add worktree at HEAD in detached mode.
	cmd := exec.Command("git", "worktree", "add", "--detach", wt, "HEAD")
	cmd.Dir = repo
	require.NoError(t, cmd.Run(), "git worktree add --detach failed")

	wts, err := ListWorktrees(repo)
	require.NoError(t, err)

	var found *Worktree
	for i := range wts {
		if resolveSymlinks(wts[i].Path) == resolveSymlinks(wt) {
			found = &wts[i]
			break
		}
	}
	require.NotNil(t, found, "detached worktree should appear in list")
	assert.Empty(t, found.Branch, "detached worktree should have empty Branch")
	assert.NotEmpty(t, found.Head)
}

func TestListWorktrees_NonRepo(t *testing.T) {
	notARepo := t.TempDir()
	_, err := ListWorktrees(notARepo)
	require.Error(t, err)
}

// seedInitialCommit ensures the repo has at least one commit so worktree-add can succeed.
func seedInitialCommit(t *testing.T, repoDir string) {
	t.Helper()
	createFile(t, repoDir, "README.md", "# seed\n")
	gitAdd(t, repoDir, "README.md")
	gitCommit(t, repoDir, "seed: initial commit")
}

// gitWorktreeAdd runs `git worktree add -b <branch> <path>` against repoDir.
func gitWorktreeAdd(t *testing.T, repoDir, wtPath, branch string) {
	t.Helper()
	cmd := exec.Command("git", "worktree", "add", "-b", branch, wtPath)
	cmd.Dir = repoDir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git worktree add failed: %s", string(out))
}
