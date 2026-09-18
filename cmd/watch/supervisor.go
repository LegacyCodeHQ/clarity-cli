package watch

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/LegacyCodeHQ/clarity/cmd/show/formatters"
	"github.com/LegacyCodeHQ/clarity/cmd/watch/protocol"
	"github.com/LegacyCodeHQ/clarity/vcs/git"
	"github.com/fsnotify/fsnotify"
)

// worktreeReconcileInterval backs up fsnotify delivery with a lightweight
// registry scan so stale worktree entries or dropped events do not strand the
// watch process. fsnotify is the fast path (instant), so this is only a
// backstop for missed events (e.g. a kqueue subscription dropped across
// sleep/wake) and runs at a relaxed cadence -- worktree adds/removes are rare.
const worktreeReconcileInterval = 2 * time.Second

// requireMainWorktree returns the canonical root of the main worktree
// containing cwd, or an error if cwd is inside a linked worktree instead.
// clarity watch must always be launched from the main worktree (see CLR-91):
// a single process discovers and watches every worktree of the repo
// together, so a launch from inside a linked worktree would otherwise either
// scope itself to just that one tree, or silently collide with a process
// already covering it from the main worktree — two processes independently
// persisting the same worktree's session history with no coordination.
func requireMainWorktree(cwd string) (string, error) {
	kind, err := git.WorktreeKindFor(cwd)
	if err != nil {
		return "", err
	}
	if kind != git.WorktreeKindMain {
		return "", fmt.Errorf("clarity watch must be run from the repository's main worktree, not a linked one; " +
			"run it from there instead — this worktree will appear automatically as a tab\n")
	}
	return git.GetWorktreeRoot(cwd)
}

// planInitialWorktrees resolves which worktrees to watch when `clarity watch`
// starts in `cwd`. The first entry is always the main worktree, given the
// literal id "main" so it's the default tab, followed by a descriptor for
// every linked worktree.
func planInitialWorktrees(cwd string) ([]protocol.WorktreeDescriptor, error) {
	root, err := requireMainWorktree(cwd)
	if err != nil {
		return nil, err
	}

	worktrees, err := git.ListWorktrees(root)
	if err != nil {
		return nil, err
	}

	descriptors := []protocol.WorktreeDescriptor{{
		ID:     mainWorktreeID,
		Path:   root,
		Label:  mainRepoLabel(root, mainBranch(worktrees)),
		Kind:   protocol.WorktreeKindMain,
		Active: true,
	}}
	for _, w := range worktrees {
		if w.Kind == git.WorktreeKindMain {
			continue
		}
		if !pathExists(w.Path) {
			continue
		}
		descriptors = append(descriptors, descriptorForLinked(w))
	}
	return descriptors, nil
}

func descriptorForLinked(w git.Worktree) protocol.WorktreeDescriptor {
	return protocol.WorktreeDescriptor{
		ID:     worktreeIDFor(w.Path, protocol.WorktreeKindLinked),
		Path:   w.Path,
		Label:  linkedRepoLabel(w.Path),
		Kind:   protocol.WorktreeKindLinked,
		Active: true,
	}
}

func mainBranch(worktrees []git.Worktree) string {
	for _, w := range worktrees {
		if w.Kind == git.WorktreeKindMain {
			return w.Branch
		}
	}
	return ""
}

// runSupervisor is the multi-worktree replacement for the old single-call
// `watchAndRebuild` flow. It registers initial tabs with the broker, fans out
// one watcher goroutine per tree, and installs a meta-watcher on
// `<common-git-dir>/worktrees/` to pick up `git worktree add` and
// `git worktree remove` events live.
//
// Returns when ctx is cancelled.
func runSupervisor(ctx context.Context, cwd string, opts *watchOptions, b *broker, formatter formatters.Formatter) error {
	descriptors, err := planInitialWorktrees(cwd)
	if err != nil {
		return err
	}

	sup := &supervisor{
		b:         b,
		opts:      opts,
		formatter: formatter,
		rootPath:  descriptors[0].Path,
		watchers:  make(map[string]context.CancelFunc),
	}

	// Install the meta-watcher BEFORE spawning initial watchers so a
	// `git worktree add` racing with startup is never missed.
	var metaDone <-chan struct{}
	// Meta-watching is best-effort; if the common dir can't be resolved or the
	// watcher fails, fall back to running without it.
	if commonDir, err := git.GetCommonDir(cwd); err == nil {
		sup.commonDir = commonDir
		ready := make(chan struct{})
		done := make(chan struct{})
		go func() {
			_ = sup.runMetaWatcher(ctx, ready)
			close(done)
		}()
		<-ready
		metaDone = done
	}

	for _, desc := range descriptors {
		sup.spawnWatcher(ctx, desc)
	}

	<-ctx.Done()
	sup.shutdown()
	if metaDone != nil {
		<-metaDone
	}
	return nil
}

type supervisor struct {
	b         *broker
	opts      *watchOptions
	formatter formatters.Formatter
	commonDir string
	rootPath  string

	mu       sync.Mutex
	watchers map[string]context.CancelFunc // worktreeID -> cancel
}

func (s *supervisor) spawnWatcher(parent context.Context, desc protocol.WorktreeDescriptor) {
	if !pathExists(desc.Path) {
		return
	}
	s.b.registerWorktree(desc)
	wctx, cancel := context.WithCancel(parent)
	s.mu.Lock()
	s.watchers[desc.ID] = cancel
	s.mu.Unlock()
	go func() {
		if err := watchAndRebuild(wctx, desc.ID, desc.Path, s.opts, s.b, s.formatter); err != nil {
			if !pathExists(desc.Path) {
				s.finishWatcher(desc.ID)
				return
			}
			fmt.Fprintf(os.Stderr, "watcher %s exited: %v\n", desc.ID, err)
		}
	}()
	// Seed an initial graph so the tab has content immediately.
	if !pathExists(desc.Path) {
		s.finishWatcher(desc.ID)
		return
	}
	publishCurrentGraph(desc.ID, desc.Path, s.opts, s.b, s.formatter)
}

// finishWatcher stops monitoring a worktree whose git working tree was removed
// but keeps its tab: the file watcher is cancelled while the broker flips the
// tab to inactive and preserves its snapshot history. The tab survives as a
// frozen, read-only record until the user closes it (see broker.closeWorktree).
func (s *supervisor) finishWatcher(worktreeID string) {
	s.mu.Lock()
	cancel, ok := s.watchers[worktreeID]
	delete(s.watchers, worktreeID)
	s.mu.Unlock()
	if ok {
		cancel()
	}
	s.b.markWorktreeFinished(worktreeID)
}

func (s *supervisor) shutdown() {
	s.mu.Lock()
	cancels := make([]context.CancelFunc, 0, len(s.watchers))
	for _, c := range s.watchers {
		cancels = append(cancels, c)
	}
	s.watchers = make(map[string]context.CancelFunc)
	s.mu.Unlock()
	for _, c := range cancels {
		c()
	}
}

// runMetaWatcher installs an fsnotify watch on `<common>/worktrees/`. Any event
// there — `git worktree add`, `remove`, or `move` — is treated as a bare trigger
// to reconcile: the watch carries no per-event logic, since `git worktree list`
// (via reconcileWorktrees) is the single source of truth for the tab set. A
// periodic tick backstops dropped fsnotify events.
// `ready` is closed once the fsnotify watch is in place, so callers can
// sequence work that must not race with the watch installation.
func (s *supervisor) runMetaWatcher(ctx context.Context, ready chan<- struct{}) error {
	closeReady := func() {
		if ready != nil {
			close(ready)
			ready = nil
		}
	}
	defer closeReady()

	worktreesDir := filepath.Join(s.commonDir, "worktrees")
	if err := os.MkdirAll(worktreesDir, 0o755); err != nil && !errors.Is(err, fs.ErrExist) {
		return fmt.Errorf("ensure worktrees dir: %w", err)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("meta-watcher create: %w", err)
	}
	defer watcher.Close()

	if err := watcher.Add(worktreesDir); err != nil {
		return fmt.Errorf("meta-watcher add %s: %w", worktreesDir, err)
	}

	// The watch is installed; let startup proceed.
	closeReady()
	reconcileTicker := time.NewTicker(worktreeReconcileInterval)
	defer reconcileTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case _, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			s.reconcileWorktrees(ctx)
		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			// kqueue reports the worktrees dir being removed (e.g. the last
			// linked worktree was pruned) as a benign ENOENT; reconcile handles
			// the actual teardown, so don't spam stderr.
			if isMissingPath(err) {
				slog.Debug("meta-watched path removed", "error", err)
				continue
			}
			fmt.Fprintf(os.Stderr, "meta-watcher error: %v\n", err)
		case <-reconcileTicker.C:
			s.reconcileWorktrees(ctx)
		}
	}
}

func (s *supervisor) reconcileWorktrees(ctx context.Context) {
	worktrees, err := git.ListWorktrees(s.rootPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "worktree reconciliation error: %v\n", err)
		return
	}

	seen := make(map[string]bool)
	for _, w := range worktrees {
		if w.Kind == git.WorktreeKindMain || !pathExists(w.Path) {
			continue
		}
		desc := descriptorForLinked(w)
		seen[desc.ID] = true
		if !s.hasWatcher(desc.ID) {
			s.spawnWatcher(ctx, desc)
		}
	}

	for _, worktreeID := range s.linkedWatchersMissingFrom(seen) {
		s.finishWatcher(worktreeID)
	}
}

func (s *supervisor) hasWatcher(worktreeID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.watchers[worktreeID] != nil
}

func (s *supervisor) linkedWatchersMissingFrom(seen map[string]bool) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var missing []string
	for worktreeID := range s.watchers {
		if worktreeID != mainWorktreeID && !seen[worktreeID] {
			missing = append(missing, worktreeID)
		}
	}
	return missing
}
