package protocol

import "time"

const (
	RouteIndex  = "/"
	RouteEvents = "/events"
	// RouteCloseWorktree tears down a finished (inactive) worktree tab. The
	// {id} path value is the WorktreeDescriptor ID. This is a
	// client→server message in the watch protocol; everything else flows
	// server→client.
	RouteCloseWorktree = "POST /worktrees/{id}/close"
	// RouteListSessions returns the metadata-only listing of every
	// persisted session across the whole project (see
	// PersistedSessionSummary) -- no snapshot content, cheap enough for
	// the client to fetch unconditionally on attach. No {id}: a single
	// `clarity watch` process only ever serves one project (CLR-91), the
	// same reason RouteIndex/RouteEvents carry none either -- the process
	// itself is the scope. A session's worktree_id is inherently
	// project-wide information (one CLR-97 watch_runs row already spans
	// every worktree the process watches), so scoping this listing to one
	// worktree would force the client to fetch per-worktree and stitch
	// runs back together itself.
	RouteListSessions = "GET /sessions"
	// RouteGetSession fetches one persisted session's full content
	// (snapshots and commits) on demand -- see PersistedSessionDetail.
	// The {id} path value is the session's database id, already globally
	// unique, so this needs no worktree scoping either.
	RouteGetSession = "GET /sessions/{id}"
)

const SSEEventGraph = "graph"

// WorktreeKind classifies a worktree using git's own vocabulary: the single
// main worktree created by `git init`/`git clone`, or one of possibly several
// linked worktrees created via `git worktree add`. See git-worktree(1). Kept
// independent of vcs/git.WorktreeKind so this package stays a self-contained
// wire contract.
type WorktreeKind string

const (
	WorktreeKindMain   WorktreeKind = "main"
	WorktreeKindLinked WorktreeKind = "linked"
)

// WorktreeDescriptor names a working tree visible to the watch session.
// One descriptor per tab in the viewer.
type WorktreeDescriptor struct {
	// ID is a stable identifier for the tree across reconnects.
	// "main" for the repository's main worktree; "wt-<hash8>" otherwise.
	ID string `json:"id"`
	// Path is the absolute path to the working tree on disk.
	Path string `json:"path"`
	// Label is a human-readable name for tab display.
	Label string `json:"label"`
	// Kind is WorktreeKindMain for the repository's main worktree,
	// WorktreeKindLinked otherwise.
	Kind WorktreeKind `json:"kind"`
	// Active reports whether the worktree is still being watched. It flips to
	// false when the underlying git worktree is removed: the tab stays visible
	// as a frozen, read-only record and becomes user-closable. The main
	// worktree stays active for the life of the watch session.
	Active bool `json:"active"`
}

// GraphSnapshot is the atom in the watch protocol timeline.
type GraphSnapshot struct {
	ID         int64     `json:"id"`
	WorktreeID string    `json:"worktreeId"`
	Timestamp  time.Time `json:"timestamp"`
	DOT        string    `json:"dot"`
	// SessionStart marks the first snapshot recorded for a worktree in this
	// watch session — the state that already existed when the watcher attached.
	// Changes made before this point are not in the timeline. Set once per repo
	// and preserved across commit/archive cycles.
	SessionStart bool `json:"sessionStart,omitempty"`
}

// CommitSummary describes a commit that caused a working snapshot cycle to
// freeze into an archived collection.
type CommitSummary struct {
	Hash      string    `json:"hash"`
	ShortHash string    `json:"shortHash"`
	Subject   string    `json:"subject"`
	Author    string    `json:"author"`
	Email     string    `json:"email"`
	Timestamp time.Time `json:"timestamp"`
}

// GraphStreamPayload is the wire payload for SSE "graph" events.
type GraphStreamPayload struct {
	Worktrees []WorktreeDescriptor `json:"worktrees"`
	// Format is the render format of every snapshot's DOT field for this watch
	// session ("dot" or "mermaid"). Session-global; clients treat an empty value
	// as "dot".
	Format                 string               `json:"format,omitempty"`
	WorkingSnapshots       []GraphSnapshot      `json:"workingSnapshots"`
	PastCollections        []SnapshotCollection `json:"pastCollections"`
	LatestWorkingID        int64                `json:"latestWorkingId"`
	LatestPastCollectionID int64                `json:"latestPastCollectionId"`
}

// SnapshotCollection represents an archived batch of working snapshots.
type SnapshotCollection struct {
	ID            int64           `json:"id"`
	WorktreeID    string          `json:"worktreeId"`
	Timestamp     time.Time       `json:"timestamp"`
	Snapshots     []GraphSnapshot `json:"snapshots"`
	CommitHistory []CommitSummary `json:"commitHistory,omitempty"`
}

// PersistedSessionSummary is one entry in the eager, metadata-only
// listing of every persisted session across the project (RouteListSessions)
// -- deliberately no snapshot content, cheap enough to send unconditionally
// on attach. RunID (CLR-97) groups sessions by which `clarity watch`
// invocation produced them, independent of which WorktreeID they belong to
// (one run can span several worktrees). ClosedAt/ClosedReason are unset for
// a still-open session.
type PersistedSessionSummary struct {
	ID            int64      `json:"id"`
	WorktreeID    string     `json:"worktreeId"`
	RunID         int64      `json:"runId"`
	Number        int        `json:"number"`
	CreatedAt     time.Time  `json:"createdAt"`
	ClosedAt      *time.Time `json:"closedAt,omitempty"`
	ClosedReason  string     `json:"closedReason,omitempty"`
	SnapshotCount int        `json:"snapshotCount"`
	CommitCount   int        `json:"commitCount"`
}

// PersistedSnapshot is one recorded snapshot within a persisted session's
// timeline, as returned by RouteGetSession. Distinct from GraphSnapshot,
// which is the live, in-memory streaming shape (a process-local auto-
// incrementing ID, no notion of database position) -- this is a durable
// database row instead.
type PersistedSnapshot struct {
	Position  int       `json:"position"`
	Source    string    `json:"source"`
	Format    string    `json:"format"`
	Kind      string    `json:"kind"`
	CreatedAt time.Time `json:"createdAt"`
}

// PersistedCommit is one commit recorded against a session closed as
// committed, as returned by RouteGetSession. Distinct from CommitSummary
// (the live worktree-removal/archive-cycle shape, which carries author/
// email/full timestamp) -- only position, hash, and subject are ever
// persisted (see cmd/watch/store.CommitRecord).
type PersistedCommit struct {
	Position int    `json:"position"`
	Hash     string `json:"hash"`
	Subject  string `json:"subject"`
}

// PersistedSessionDetail is the full content of one persisted session,
// fetched on demand via RouteGetSession only when the user actually clicks
// into it -- see PersistedSessionSummary for the eager listing every other
// session is shown through.
type PersistedSessionDetail struct {
	PersistedSessionSummary
	Snapshots []PersistedSnapshot `json:"snapshots"`
	Commits   []PersistedCommit   `json:"commits,omitempty"`
}
