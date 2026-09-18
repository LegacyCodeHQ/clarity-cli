package watch

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/LegacyCodeHQ/clarity/cmd/watch/protocol"
	"github.com/LegacyCodeHQ/clarity/cmd/watch/store"
	"github.com/LegacyCodeHQ/clarity/vcs"
)

const maxSnapshots = 250

const watchPageTitleSuffix = "clarity watch"

// worktreeState holds the per-worktree snapshot history and archived cycles.
type worktreeState struct {
	history        []protocol.GraphSnapshot
	archivedCycles []protocol.SnapshotCollection
	hasState       bool
	// sessionStarted records whether this worktree's first snapshot (the
	// watcher-attach state) has already been emitted. Set once and never reset,
	// so the session-start marker doesn't reappear after commit/archive cycles.
	sessionStarted bool
	// dbSessionID and dbSnapshotPos track the persisted session this
	// process's live snapshot stream is being written into (see
	// broker.publish). dbSessionID is 0 until the first snapshot of this
	// worktree's lifetime in this process opens one via store.OpenSession —
	// each process always opens a fresh session, never resumes one left
	// open by an earlier run (that's restart hydration's job, not this).
	dbSessionID   int64
	dbSnapshotPos int
}

// broker manages SSE client connections and broadcasts graph snapshots.
// It maintains independent snapshot history per registered worktree (`worktreeID`)
// while aggregating them into a single flat SSE payload.
type broker struct {
	mu             sync.Mutex
	clients        map[chan protocol.GraphStreamPayload]struct{}
	worktrees      []protocol.WorktreeDescriptor
	worktreeIndex  map[string]int
	worktreeStates map[string]*worktreeState
	nextID         int64
	nextCycleID    int64
	// format is the session-global render format ("dot" or "mermaid") echoed to
	// clients so the viewer knows how to render each snapshot's DOT payload. An
	// empty value is treated as "dot".
	format string
	// dbStore and projectID enable persisting worktree lifecycle events
	// (see enablePersistence). dbStore is nil by default, which makes every
	// persistence call in this file a no-op — existing tests constructing a
	// broker with newBroker() are unaffected.
	dbStore   *sql.DB
	projectID string
}

func newBroker() *broker {
	return &broker{
		clients:        make(map[chan protocol.GraphStreamPayload]struct{}),
		worktreeIndex:  make(map[string]int),
		worktreeStates: make(map[string]*worktreeState),
	}
}

// enablePersistence turns on shadow-writing worktree lifecycle events
// (registration, disposal, hiding) to db under projectID. It must be called
// before any worktree is registered. Persistence failures are logged, never
// fatal — clarity watch's live behavior does not depend on the database.
func (b *broker) enablePersistence(db *sql.DB, projectID string) {
	b.dbStore = db
	b.projectID = projectID
}

func (b *broker) subscribe() chan protocol.GraphStreamPayload {
	ch := make(chan protocol.GraphStreamPayload, 1)
	b.mu.Lock()
	b.clients[ch] = struct{}{}
	payload, ok := b.currentPayloadLocked()
	if ok {
		ch <- payload
	}
	b.mu.Unlock()
	return ch
}

func (b *broker) unsubscribe(ch chan protocol.GraphStreamPayload) {
	b.mu.Lock()
	delete(b.clients, ch)
	close(ch)
	b.mu.Unlock()
}

// registerWorktree adds a worktree to the broker's tab set. If `desc.ID`
// already exists, the descriptor is updated in place (path/label/kind
// may change on git operations like `worktree move`).
func (b *broker) registerWorktree(desc protocol.WorktreeDescriptor) {
	b.mu.Lock()
	if idx, ok := b.worktreeIndex[desc.ID]; ok {
		b.worktrees[idx] = desc
	} else {
		b.worktreeIndex[desc.ID] = len(b.worktrees)
		b.worktrees = append(b.worktrees, desc)
		b.worktreeStates[desc.ID] = &worktreeState{}
	}
	b.broadcastLocked()
	dbStore, projectID := b.dbStore, b.projectID
	b.mu.Unlock()

	if dbStore != nil {
		if err := store.RegisterWorktree(dbStore, desc.ID, projectID, desc.Path, desc.Kind, desc.Label); err != nil {
			fmt.Fprintf(os.Stderr, "persist worktree %s: %v\n", desc.ID, err)
		}
	}
}

// unregisterWorktree removes a worktree and its snapshot history outright.
// Used on shutdown; the live `git worktree remove` path goes through
// markWorktreeFinished instead so the tab lingers as a closable record.
func (b *broker) unregisterWorktree(worktreeID string) {
	b.mu.Lock()
	if idx, ok := b.worktreeIndex[worktreeID]; ok {
		b.unregisterWorktreeLocked(idx, worktreeID)
		b.broadcastLocked()
	}
	b.mu.Unlock()
}

// unregisterWorktreeLocked drops a worktree from the tab set and deletes its
// state. The caller must hold b.mu and is responsible for broadcasting.
func (b *broker) unregisterWorktreeLocked(idx int, worktreeID string) {
	b.worktrees = append(b.worktrees[:idx], b.worktrees[idx+1:]...)
	delete(b.worktreeIndex, worktreeID)
	delete(b.worktreeStates, worktreeID)
	for id, i := range b.worktreeIndex {
		if i > idx {
			b.worktreeIndex[id] = i - 1
		}
	}
}

// markWorktreeFinished flips a worktree to inactive when its git working
// tree is removed. The tab and its snapshot history are KEPT so the user can
// still browse the frozen final state; the UI surfaces a close affordance
// and the teardown completes via closeWorktree.
func (b *broker) markWorktreeFinished(worktreeID string) {
	b.mu.Lock()
	finished := false
	var dbSessionID int64
	var hadHistory bool
	if idx, ok := b.worktreeIndex[worktreeID]; ok && b.worktrees[idx].Active {
		s := b.stateForLocked(worktreeID)
		dbSessionID, hadHistory = b.archiveWorkingSetLocked(worktreeID, s, nil)
		b.worktrees[idx].Active = false
		b.broadcastLocked()
		finished = true
	}
	dbStore := b.dbStore
	b.mu.Unlock()

	if !finished || dbStore == nil {
		return
	}

	if err := store.MarkWorktreeDisposed(dbStore, worktreeID); err != nil {
		fmt.Fprintf(os.Stderr, "persist worktree %s disposed: %v\n", worktreeID, err)
	}
	// A session can still have been open (uncommitted snapshots in flight)
	// when the worktree disappeared — distinct from the user reverting
	// while still watching, hence its own closed_reason.
	if hadHistory && dbSessionID != 0 {
		if err := store.CloseSessionWorktreeRemoved(dbStore, dbSessionID); err != nil {
			fmt.Fprintf(os.Stderr, "persist session close (worktree_removed) for %s: %v\n", worktreeID, err)
		}
	}
}

// closeOutcome reports how a closeWorktree request resolved, letting the
// HTTP layer map it to a status code without leaking transport concerns into
// the broker.
type closeOutcome int

const (
	closeOK closeOutcome = iota
	closeNotFound
	closeActive
)

// closeWorktree tears down a finished tab at the user's request. Active
// worktrees are pinned: they cannot be closed while still being watched.
func (b *broker) closeWorktree(worktreeID string) closeOutcome {
	b.mu.Lock()
	idx, ok := b.worktreeIndex[worktreeID]
	if !ok {
		b.mu.Unlock()
		return closeNotFound
	}
	if b.worktrees[idx].Active {
		b.mu.Unlock()
		return closeActive
	}
	b.unregisterWorktreeLocked(idx, worktreeID)
	b.broadcastLocked()
	dbStore := b.dbStore
	b.mu.Unlock()

	if dbStore != nil {
		if err := store.MarkWorktreeHidden(dbStore, worktreeID); err != nil {
			fmt.Fprintf(os.Stderr, "persist worktree %s hidden: %v\n", worktreeID, err)
		}
	}
	return closeOK
}

// stateForLocked returns the per-worktree state, creating it for an
// unregistered worktreeID. This keeps tests and single-worktree callers
// ergonomic; the supervisor will registerWorktree first in real flows.
func (b *broker) stateForLocked(worktreeID string) *worktreeState {
	if s, ok := b.worktreeStates[worktreeID]; ok {
		return s
	}
	s := &worktreeState{}
	b.worktreeStates[worktreeID] = s
	return s
}

func (b *broker) publish(worktreeID, dot string) {
	b.mu.Lock()
	s := b.stateForLocked(worktreeID)
	if len(s.history) > 0 && s.history[len(s.history)-1].DOT == dot {
		b.mu.Unlock()
		return
	}

	b.nextID++
	sessionStart := !s.sessionStarted
	s.sessionStarted = true
	timestamp := time.Now().UTC()
	s.history = append(s.history, protocol.GraphSnapshot{
		ID:           b.nextID,
		WorktreeID:   worktreeID,
		Timestamp:    timestamp,
		DOT:          dot,
		SessionStart: sessionStart,
	})
	if len(s.history) > maxSnapshots {
		s.history = s.history[len(s.history)-maxSnapshots:]
	}
	s.hasState = true

	b.broadcastLocked()
	dbStore, format := b.dbStore, b.format
	needsNewSession := dbStore != nil && s.dbSessionID == 0
	b.mu.Unlock()

	if dbStore == nil {
		return
	}
	b.persistSnapshot(dbStore, worktreeID, s, needsNewSession, dot, format, sessionStart, timestamp)
}

// persistSnapshot writes one snapshot to the database, opening a session
// first if this is the first snapshot of this worktree's lifetime in this
// process. Failures are logged, never fatal — clarity watch's live
// behavior does not depend on the database.
func (b *broker) persistSnapshot(
	dbStore *sql.DB, worktreeID string, s *worktreeState,
	needsNewSession bool, dot, format string, sessionStart bool, timestamp time.Time,
) {
	if needsNewSession {
		sessionID, err := store.OpenSession(dbStore, worktreeID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "open persisted session for %s: %v\n", worktreeID, err)
			return
		}
		b.mu.Lock()
		s.dbSessionID = sessionID
		s.dbSnapshotPos = 0
		b.mu.Unlock()
	}

	b.mu.Lock()
	sessionID := s.dbSessionID
	position := s.dbSnapshotPos
	s.dbSnapshotPos++
	b.mu.Unlock()

	if sessionID == 0 {
		return // opening the session above already failed and was logged
	}

	if format == "" {
		format = "dot"
	}
	kind := "incremental"
	if sessionStart {
		kind = "baseline"
	}
	if err := store.AppendSnapshot(dbStore, sessionID, position, dot, format, kind, timestamp); err != nil {
		fmt.Fprintf(os.Stderr, "persist snapshot for %s: %v\n", worktreeID, err)
	}
}

func (b *broker) archiveWorkingSet(worktreeID string) {
	b.mu.Lock()
	s := b.stateForLocked(worktreeID)
	b.archiveWorkingSetLocked(worktreeID, s, nil)
	b.broadcastLocked()
	b.mu.Unlock()
}

// archiveWorkingSetWithCommitHistory closes the current session on a real
// commit. A HEAD change with no reachable commits (git reset --hard, amend,
// rebase discarding work) is not a new session — it's a no-op here, and the
// working set stays open so the next real commit picks it back up.
func (b *broker) archiveWorkingSetWithCommitHistory(worktreeID string, commitHistory []vcs.CommitSummary) {
	if len(commitHistory) == 0 {
		return
	}
	b.mu.Lock()
	s := b.stateForLocked(worktreeID)
	dbSessionID, hadHistory := b.archiveWorkingSetLocked(worktreeID, s, commitHistory)
	b.broadcastLocked()
	dbStore := b.dbStore
	b.mu.Unlock()

	if !hadHistory || dbStore == nil || dbSessionID == 0 {
		return
	}
	commits := make([]store.CommitRecord, len(commitHistory))
	for i, c := range commitHistory {
		commits[i] = store.CommitRecord{Position: i, Hash: c.Hash, Subject: c.Subject}
	}
	if err := store.CloseSessionCommitted(dbStore, dbSessionID, commits); err != nil {
		fmt.Fprintf(os.Stderr, "persist session close (committed) for %s: %v\n", worktreeID, err)
	}
}

// archiveWorkingSetLocked returns the persisted session id that was open
// (0 if none) and whether there was anything to archive. The caller — which
// holds b.mu across this call — must release the lock before actually
// persisting the close, since it's an I/O call.
func (b *broker) archiveWorkingSetLocked(worktreeID string, s *worktreeState, commitHistory []vcs.CommitSummary) (dbSessionID int64, hadHistory bool) {
	if len(s.history) > 0 {
		archivedSnapshots := make([]protocol.GraphSnapshot, len(s.history))
		copy(archivedSnapshots, s.history)
		b.nextCycleID++
		s.archivedCycles = append(s.archivedCycles, protocol.SnapshotCollection{
			ID:            b.nextCycleID,
			WorktreeID:    worktreeID,
			Timestamp:     time.Now().UTC(),
			Snapshots:     archivedSnapshots,
			CommitHistory: toProtocolCommitHistory(commitHistory),
		})
		dbSessionID = s.dbSessionID
		hadHistory = true
	}

	s.history = nil
	s.dbSessionID = 0
	s.dbSnapshotPos = 0
	s.hasState = true
	return dbSessionID, hadHistory
}

func (b *broker) clearWorkingSet(worktreeID string) {
	b.mu.Lock()
	s := b.stateForLocked(worktreeID)
	if len(s.history) == 0 && s.hasState {
		b.mu.Unlock()
		return
	}

	dbSessionID := s.dbSessionID
	hadHistory := len(s.history) > 0

	s.history = nil
	s.dbSessionID = 0
	s.dbSnapshotPos = 0
	s.hasState = true
	b.broadcastLocked()
	dbStore := b.dbStore
	b.mu.Unlock()

	if hadHistory && dbStore != nil && dbSessionID != 0 {
		if err := store.CloseSessionDiscarded(dbStore, dbSessionID); err != nil {
			fmt.Fprintf(os.Stderr, "persist session close (discarded) for %s: %v\n", worktreeID, err)
		}
	}
}

func (b *broker) broadcastLocked() {
	payload, ok := b.currentPayloadLocked()
	if !ok {
		return
	}
	for ch := range b.clients {
		pushLatestPayload(ch, payload)
	}
}

func (b *broker) currentPayloadLocked() (protocol.GraphStreamPayload, bool) {
	anyHasState := false
	for _, s := range b.worktreeStates {
		if s.hasState {
			anyHasState = true
			break
		}
	}
	if !anyHasState && len(b.worktrees) == 0 {
		return protocol.GraphStreamPayload{}, false
	}

	worktrees := make([]protocol.WorktreeDescriptor, len(b.worktrees))
	copy(worktrees, b.worktrees)

	working := b.collectWorkingLocked()
	past := b.collectPastLocked()

	var latestWorkingID int64
	for _, snap := range working {
		if snap.ID > latestWorkingID {
			latestWorkingID = snap.ID
		}
	}
	var latestPastID int64
	for _, coll := range past {
		if coll.ID > latestPastID {
			latestPastID = coll.ID
		}
	}

	return protocol.GraphStreamPayload{
		Worktrees:              worktrees,
		Format:                 b.format,
		WorkingSnapshots:       working,
		PastCollections:        past,
		LatestWorkingID:        latestWorkingID,
		LatestPastCollectionID: latestPastID,
	}, true
}

// collectWorkingLocked returns a flat, deterministic list of working
// snapshots across every worktree: outer order follows worktree registration
// order; inner order preserves per-worktree history order.
func (b *broker) collectWorkingLocked() []protocol.GraphSnapshot {
	worktreeIDs := b.orderedWorktreeIDsLocked()
	working := []protocol.GraphSnapshot{}
	for _, id := range worktreeIDs {
		s := b.worktreeStates[id]
		if s == nil {
			continue
		}
		working = append(working, s.history...)
	}
	return working
}

func (b *broker) collectPastLocked() []protocol.SnapshotCollection {
	worktreeIDs := b.orderedWorktreeIDsLocked()
	past := []protocol.SnapshotCollection{}
	for _, id := range worktreeIDs {
		s := b.worktreeStates[id]
		if s == nil {
			continue
		}
		for _, cycle := range s.archivedCycles {
			snapshots := make([]protocol.GraphSnapshot, len(cycle.Snapshots))
			copy(snapshots, cycle.Snapshots)
			past = append(past, protocol.SnapshotCollection{
				ID:            cycle.ID,
				WorktreeID:    cycle.WorktreeID,
				Timestamp:     cycle.Timestamp,
				Snapshots:     snapshots,
				CommitHistory: copyCommitHistory(cycle.CommitHistory),
			})
		}
	}
	return past
}

func toProtocolCommitHistory(commits []vcs.CommitSummary) []protocol.CommitSummary {
	if len(commits) == 0 {
		return nil
	}
	history := make([]protocol.CommitSummary, 0, len(commits))
	for _, commit := range commits {
		history = append(history, protocol.CommitSummary{
			Hash:      commit.Hash,
			ShortHash: commit.ShortHash,
			Subject:   commit.Subject,
			Author:    commit.Author,
			Email:     commit.Email,
			Timestamp: commit.Timestamp,
		})
	}
	return history
}

func copyCommitHistory(commits []protocol.CommitSummary) []protocol.CommitSummary {
	if len(commits) == 0 {
		return nil
	}
	copied := make([]protocol.CommitSummary, len(commits))
	copy(copied, commits)
	return copied
}

// orderedWorktreeIDsLocked returns worktree IDs in registration order, then
// any orphan worktree states (registered via publish without
// registerWorktree) sorted alphabetically for determinism in tests.
func (b *broker) orderedWorktreeIDsLocked() []string {
	ids := make([]string, 0, len(b.worktreeStates))
	seen := make(map[string]bool, len(b.worktrees))
	for _, r := range b.worktrees {
		ids = append(ids, r.ID)
		seen[r.ID] = true
	}
	var orphans []string
	for id := range b.worktreeStates {
		if !seen[id] {
			orphans = append(orphans, id)
		}
	}
	sort.Strings(orphans)
	return append(ids, orphans...)
}

func pushLatestPayload(ch chan protocol.GraphStreamPayload, payload protocol.GraphStreamPayload) {
	select {
	case ch <- payload:
		return
	default:
	}

	select {
	case <-ch:
	default:
	}

	select {
	case ch <- payload:
	default:
	}
}

func newServer(b *broker, port int, worktreePath string) *http.Server {
	mux := http.NewServeMux()

	// Serve index.html with page title injection
	mux.HandleFunc(protocol.RouteIndex, handleIndex(buildWatchPageTitle(worktreePath)))

	// Serve all static assets from embedded dist directory
	distFS, err := getDistFS()
	if err != nil {
		panic(fmt.Sprintf("failed to get dist FS: %v", err))
	}
	mux.Handle("/assets/", http.FileServer(http.FS(distFS)))

	// Serve SSE endpoint (unchanged)
	mux.HandleFunc(protocol.RouteEvents, handleSSE(b))

	// Client→server: close a finished worktree tab.
	mux.HandleFunc(protocol.RouteCloseWorktree, handleCloseWorktree(b))

	return &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}
}

func buildWatchPageTitle(worktreePath string) string {
	worktreeName := strings.TrimSpace(filepath.Base(filepath.Clean(worktreePath)))
	if worktreeName == "" || worktreeName == "." || worktreeName == string(filepath.Separator) {
		return watchPageTitleSuffix
	}

	return fmt.Sprintf("%s • %s", worktreeName, watchPageTitleSuffix)
}

func handleIndex(pageTitle string) http.HandlerFunc {
	view := struct {
		PageTitle string
	}{
		PageTitle: pageTitle,
	}

	return func(w http.ResponseWriter, _ *http.Request) {
		// Read index.html from embedded dist directory
		distFS, err := getDistFS()
		if err != nil {
			http.Error(w, "failed to load assets", http.StatusInternalServerError)
			return
		}

		indexFile, err := distFS.Open("index.html")
		if err != nil {
			http.Error(w, "failed to load index.html", http.StatusInternalServerError)
			return
		}
		defer indexFile.Close()

		indexContent, err := io.ReadAll(indexFile)
		if err != nil {
			http.Error(w, "failed to read index.html", http.StatusInternalServerError)
			return
		}

		// Execute template to inject page title
		tmpl, err := template.New("index").Parse(string(indexContent))
		if err != nil {
			http.Error(w, "failed to parse template", http.StatusInternalServerError)
			return
		}

		var rendered bytes.Buffer
		if err := tmpl.Execute(&rendered, view); err != nil {
			http.Error(w, "failed to render page", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if _, err := w.Write(rendered.Bytes()); err != nil {
			http.Error(w, "failed to write response", http.StatusInternalServerError)
		}
	}
}

// handleCloseWorktree tears down a finished worktree tab. Active worktrees
// are pinned (409); unknown ids are 404. On success the broker broadcasts
// the updated tab set, so connected clients drop the tab via the normal SSE
// flow.
func handleCloseWorktree(b *broker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		worktreeID := r.PathValue("id")
		switch b.closeWorktree(worktreeID) {
		case closeOK:
			w.WriteHeader(http.StatusNoContent)
		case closeActive:
			http.Error(w, "worktree is still active", http.StatusConflict)
		case closeNotFound:
			http.Error(w, "unknown worktree", http.StatusNotFound)
		}
	}
}

func handleSSE(b *broker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		ch := b.subscribe()
		defer b.unsubscribe(ch)

		ctx := r.Context()
		for {
			select {
			case <-ctx.Done():
				return
			case payload, ok := <-ch:
				if !ok {
					return
				}
				body, err := json.Marshal(payload)
				if err != nil {
					continue
				}
				fmt.Fprintf(w, "event: %s\n", protocol.SSEEventGraph)
				for _, line := range strings.Split(string(body), "\n") {
					fmt.Fprintf(w, "data: %s\n", line)
				}
				fmt.Fprintf(w, "\n")
				flusher.Flush()
			}
		}
	}
}
