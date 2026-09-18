package watch

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/LegacyCodeHQ/clarity/cmd/show/formatters"
	"github.com/LegacyCodeHQ/clarity/cmd/watch/protocol"
	"github.com/LegacyCodeHQ/clarity/vcs/git"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanInitialWorktrees_MainNoWorktrees(t *testing.T) {
	repo := initRepoWithCommit(t)

	descriptors, err := planInitialWorktrees(repo)
	require.NoError(t, err)
	require.Len(t, descriptors, 1)
	assert.Equal(t, mainWorktreeID, descriptors[0].ID)
	assert.Equal(t, protocol.WorktreeKindMain, descriptors[0].Kind)
	assert.Equal(t, "main", descriptors[0].Label)
}

func TestPlanInitialWorktrees_MainWithLinkedWorktree(t *testing.T) {
	repo := initRepoWithCommit(t)
	wt := filepath.Join(t.TempDir(), "linked")
	runGit(t, repo, "worktree", "add", "-b", "feat/x", wt)

	descriptors, err := planInitialWorktrees(repo)
	require.NoError(t, err)
	require.Len(t, descriptors, 2)

	assert.Equal(t, mainWorktreeID, descriptors[0].ID)
	assert.Equal(t, protocol.WorktreeKindMain, descriptors[0].Kind)
	assert.Equal(t, "main", descriptors[0].Label)
	// The linked worktree comes after the main worktree, with a derived id.
	assert.True(t, descriptors[1].ID != mainWorktreeID, "linked worktree should not get the main id")
	assert.Equal(t, protocol.WorktreeKindLinked, descriptors[1].Kind)
	assert.Equal(t, "linked", descriptors[1].Label, "label should be the worktree directory name")
}

// TestPlanInitialWorktrees_SubdirectoryOfMainStillResolvesToRoot guards the
// canonicalization fix from CLR-91: launching from a subdirectory of the
// main worktree must resolve to the same root — and so the same "main"
// identity and full worktree set — as launching from the root itself,
// rather than silently narrowing the watched scope to that subdirectory.
func TestPlanInitialWorktrees_SubdirectoryOfMainStillResolvesToRoot(t *testing.T) {
	repo := initRepoWithCommit(t)
	sub := filepath.Join(repo, "sub")
	require.NoError(t, os.MkdirAll(sub, 0o755))

	descriptors, err := planInitialWorktrees(sub)
	require.NoError(t, err)
	require.Len(t, descriptors, 1)
	resolvedRepo, err := filepath.EvalSymlinks(repo)
	require.NoError(t, err)
	assert.Equal(t, resolvedRepo, descriptors[0].Path, "must resolve to the worktree root, not the launch subdirectory")
}

// TestPlanInitialWorktrees_LinkedWorktreeRejected pins the CLR-91 launch
// constraint: clarity watch must be started from the main worktree.
// Launching from inside a linked worktree is rejected rather than silently
// scoping itself to just that one tree, since a second process launched
// from the main worktree would otherwise cover the same worktree with no
// coordination between the two.
func TestPlanInitialWorktrees_LinkedWorktreeRejected(t *testing.T) {
	repo := initRepoWithCommit(t)
	wt := filepath.Join(t.TempDir(), "linked")
	runGit(t, repo, "worktree", "add", "-b", "feat/x", wt)

	_, err := planInitialWorktrees(wt)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "main worktree")
}

func TestPlanInitialWorktrees_NonRepoErrors(t *testing.T) {
	_, err := planInitialWorktrees(t.TempDir())
	require.Error(t, err)
}

// TestSupervisor_DetectsLiveWorktreeAdd is the core test for the user-facing
// behavior: starting `clarity watch` in the main tree should make a newly
// added linked worktree appear as a tab without restarting.
func TestSupervisor_DetectsLiveWorktreeAdd(t *testing.T) {
	repo := initRepoWithCommit(t)
	b := newBroker()
	formatter, err := formatters.NewFormatter("dot")
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	supervisorDone := make(chan struct{})
	go func() {
		_ = runSupervisor(ctx, repo, &watchOptions{}, b, formatter)
		close(supervisorDone)
	}()

	// Wait for the initial main tab to register.
	require.Eventually(t, func() bool {
		b.mu.Lock()
		defer b.mu.Unlock()
		return len(b.worktrees) == 1
	}, 2*time.Second, 20*time.Millisecond, "main tab should register on startup")

	// Add a worktree from outside the supervisor and expect it to appear as a tab.
	wt := filepath.Join(t.TempDir(), "live-added")
	runGit(t, repo, "worktree", "add", "-b", "feat/live", wt)

	require.Eventually(t, func() bool {
		b.mu.Lock()
		defer b.mu.Unlock()
		return len(b.worktrees) == 2
	}, 3*time.Second, 50*time.Millisecond, "supervisor should add a tab for the new worktree")

	b.mu.Lock()
	gotIDs := []string{b.worktrees[0].ID, b.worktrees[1].ID}
	linkedID := b.worktrees[1].ID
	bothActive := b.worktrees[0].Active && b.worktrees[1].Active
	b.mu.Unlock()
	assert.Contains(t, gotIDs, mainWorktreeID)
	assert.NotEqual(t, mainWorktreeID, gotIDs[1], "second tab should be the linked worktree, not another main tab")
	assert.True(t, bothActive, "freshly watched worktrees start active")

	// Removing the worktree keeps the tab as a frozen, inactive record — the
	// user closes it explicitly. The tab must NOT vanish on its own.
	runGit(t, repo, "worktree", "remove", "--force", wt)
	require.Eventually(t, func() bool {
		b.mu.Lock()
		defer b.mu.Unlock()
		if len(b.worktrees) != 2 {
			return false
		}
		idx, ok := b.worktreeIndex[linkedID]
		return ok && !b.worktrees[idx].Active
	}, 3*time.Second, 50*time.Millisecond, "removed worktree should remain as an inactive tab")

	cancel()
	select {
	case <-supervisorDone:
	case <-time.After(2 * time.Second):
		t.Fatal("supervisor did not exit after cancel")
	}
}

// TestSupervisor_ReconcileDiscoversWorktreeWithoutFsnotify pins the reconcile
// backstop in isolation: with no meta-watcher running, a worktree added after
// the supervisor exists is invisible to fsnotify, so only reconcileWorktrees can
// discover it. This guards the recovery path for a dropped/stale fsnotify
// subscription (the failure that strands a long-running watch) and would fail if
// the reconcile scan were removed.
func TestSupervisor_ReconcileDiscoversWorktreeWithoutFsnotify(t *testing.T) {
	repo := initRepoWithCommit(t)
	b := newBroker()
	formatter, err := formatters.NewFormatter("dot")
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s := &supervisor{
		b:         b,
		opts:      &watchOptions{},
		formatter: formatter,
		rootPath:  repo,
		watchers:  make(map[string]context.CancelFunc),
	}

	// Added with no fsnotify meta-watcher in the picture, so only an explicit
	// reconcile can discover it.
	live := filepath.Join(t.TempDir(), "reconciled")
	runGit(t, repo, "worktree", "add", "-b", "feat/reconciled", live)

	countReconciled := func() int {
		b.mu.Lock()
		defer b.mu.Unlock()
		n := 0
		for _, r := range b.worktrees {
			if r.Label == "reconciled" && r.Active {
				n++
			}
		}
		return n
	}

	require.Zero(t, countReconciled(), "tab must not exist before a reconcile runs")

	s.reconcileWorktrees(ctx)
	require.Eventually(t, func() bool { return countReconciled() == 1 },
		2*time.Second, 20*time.Millisecond,
		"reconcile alone should register the worktree tab")

	// Idempotent: a second pass must not double-register the same worktree.
	s.reconcileWorktrees(ctx)
	require.Equal(t, 1, countReconciled(), "reconcile must be idempotent")
}

// TestSupervisor_SkipsStaleInitialWorktreeAndDetectsLaterAdds reproduces the
// failure mode where Git still lists a linked worktree after its directory has
// disappeared. A stale startup entry must not leave a dead active tab or break
// discovery of subsequently added worktrees.
func TestSupervisor_SkipsStaleInitialWorktreeAndDetectsLaterAdds(t *testing.T) {
	repo := initRepoWithCommit(t)
	stale := filepath.Join(t.TempDir(), "stale")
	runGit(t, repo, "worktree", "add", "-b", "feat/stale", stale)
	require.NoError(t, os.RemoveAll(stale))

	b := newBroker()
	formatter, err := formatters.NewFormatter("dot")
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	supervisorDone := make(chan struct{})
	go func() {
		_ = runSupervisor(ctx, repo, &watchOptions{}, b, formatter)
		close(supervisorDone)
	}()

	require.Eventually(t, func() bool {
		b.mu.Lock()
		defer b.mu.Unlock()
		return len(b.worktrees) > 0 && b.worktrees[0].ID == mainWorktreeID
	}, 2*time.Second, 20*time.Millisecond, "main tab should register on startup")

	live := filepath.Join(t.TempDir(), "live-after-stale")
	runGit(t, repo, "worktree", "add", "-b", "feat/live-after-stale", live)

	require.Eventually(t, func() bool {
		b.mu.Lock()
		defer b.mu.Unlock()
		if len(b.worktrees) != 2 {
			return false
		}
		hasLive := false
		for _, worktree := range b.worktrees {
			if worktree.Label == "stale" {
				return false
			}
			if worktree.Label == "live-after-stale" && worktree.Active {
				hasLive = true
			}
		}
		return hasLive
	}, 3*time.Second, 50*time.Millisecond, "live worktree discovery should continue after a stale startup entry")

	cancel()
	select {
	case <-supervisorDone:
	case <-time.After(2 * time.Second):
		t.Fatal("supervisor did not exit after cancel")
	}
}

// TestSupervisor_VanishedWorktreeSelfFinishes guards the teardown backstop: a
// watcher whose working directory disappears must stop polling git and flip its
// tab to inactive on its own, WITHOUT a meta-watcher REMOVE event. This is the
// case that breaks when fsnotify coalesces/drops events during a batch
// `git worktree remove`, leaving watchers polling deleted directories forever.
func TestSupervisor_VanishedWorktreeSelfFinishes(t *testing.T) {
	repo := initRepoWithCommit(t)
	wt := filepath.Join(t.TempDir(), "gone")
	runGit(t, repo, "worktree", "add", "-b", "feat/gone", wt)

	b := newBroker()
	formatter, err := formatters.NewFormatter("dot")
	require.NoError(t, err)

	sup := &supervisor{
		b:         b,
		opts:      &watchOptions{},
		formatter: formatter,
		watchers:  make(map[string]context.CancelFunc),
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Spawn the watcher directly, bypassing the meta-watcher entirely.
	desc := descriptorForLinked(git.Worktree{Path: wt, Branch: "feat/gone"})
	sup.spawnWatcher(ctx, desc)

	require.Eventually(t, func() bool {
		b.mu.Lock()
		defer b.mu.Unlock()
		idx, ok := b.worktreeIndex[desc.ID]
		return ok && b.worktrees[idx].Active
	}, 2*time.Second, 20*time.Millisecond, "worktree tab should register active")

	// Delete the working tree out from under the watcher without telling the
	// supervisor — simulating a dropped REMOVE event.
	require.NoError(t, os.RemoveAll(wt))

	require.Eventually(t, func() bool {
		b.mu.Lock()
		defer b.mu.Unlock()
		idx, ok := b.worktreeIndex[desc.ID]
		return ok && !b.worktrees[idx].Active
	}, 5*time.Second, 100*time.Millisecond, "watcher should self-finish when its worktree vanishes")
}

// initRepoWithCommit creates a fresh git repo with one commit so worktree-add
// can succeed, then returns its absolute path.
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
