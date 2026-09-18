/**
 * Type definitions and normalization functions for the SSE graph stream protocol.
 * These functions validate and normalize untrusted JSON payloads from the server.
 */

// WorktreeKind classifies a worktree using git's own vocabulary: the single
// main worktree created by `git init`/`git clone`, or one of possibly several
// linked worktrees created via `git worktree add`. See git-worktree(1).
export type WorktreeKind = "main" | "linked";

export interface WorktreeDescriptor {
  id: string;
  path: string;
  label: string;
  kind: WorktreeKind;
  // False once the underlying git worktree is removed — the tab becomes a
  // frozen, closable record. A missing flag is treated as active.
  active: boolean;
}

export interface Snapshot {
  id: number;
  worktreeId?: string;
  timestamp: string;
  dot: string;
  // True for the first snapshot recorded for a worktree this session — the
  // state present when the watcher attached. Omitted (falsy) otherwise.
  sessionStart?: boolean;
}

export interface CommitSummary {
  hash: string;
  shortHash: string;
  subject: string;
  author: string;
  email: string;
  timestamp: string;
}

export interface Collection {
  id: number;
  worktreeId?: string;
  timestamp: string;
  snapshots: Snapshot[];
  commitHistory: CommitSummary[];
  // The persisted sessions.id this collection was archived into, when
  // persistence is enabled. Lets the client recognize this collection as
  // the same session that would otherwise also appear in the
  // GET /sessions listing — this run's own closed sessions must not show
  // up twice.
  sessionId?: number;
}

// PersistedSessionSummary is one entry in the metadata-only listing of a
// project's persisted session history (GET /sessions) — no snapshot
// content. See PersistedSessionDetail for the full, on-demand fetch.
export interface PersistedSessionSummary {
  id: number;
  worktreeId: string;
  runId: number;
  number: number;
  createdAt: string;
  // null for a still-open session — the current run's own live state for
  // that worktree, not something history browsing should ever show.
  closedAt: string | null;
  closedReason: string;
  snapshotCount: number;
  commitCount: number;
}

export interface PersistedSnapshot {
  position: number;
  source: string;
  format: string;
  kind: string;
  createdAt: string;
}

export interface PersistedCommit {
  position: number;
  hash: string;
  subject: string;
}

// PersistedSessionDetail is the full content of one persisted session
// (GET /sessions/{id}), fetched only once the user clicks into it.
export interface PersistedSessionDetail extends PersistedSessionSummary {
  snapshots: PersistedSnapshot[];
  commits: PersistedCommit[];
}

export interface GraphStreamPayload {
  worktrees?: WorktreeDescriptor[];
  // Session-global render format of every snapshot's `dot` field ("dot" or
  // "mermaid"). Absent on older payloads; treated as "dot".
  format?: string;
  workingSnapshots: Snapshot[];
  pastCollections: Collection[];
  latestWorkingId?: number;
  latestPastCollectionId?: number;
}

function normalizeWorktree(worktree: unknown): WorktreeDescriptor | null {
  if (!worktree || typeof worktree !== "object") {
    return null;
  }
  const r = worktree as Record<string, unknown>;
  if (typeof r.id !== "string" || r.id === "") {
    return null;
  }
  return {
    id: r.id,
    path: typeof r.path === "string" ? r.path : "",
    label: typeof r.label === "string" ? r.label : r.id,
    kind: r.kind === "main" ? "main" : "linked",
    // Treat a missing flag as active so older payloads keep their tabs pinned.
    active: r.active !== false,
  };
}

function normalizeSnapshot(snapshot: unknown): Snapshot | null {
  if (!snapshot || typeof snapshot !== "object") {
    return null;
  }
  const s = snapshot as Record<string, unknown>;
  if (typeof s.dot !== "string") {
    return null;
  }

  const normalized: Snapshot = {
    id: Number.isFinite(s.id) ? (s.id as number) : 0,
    worktreeId: typeof s.worktreeId === "string" ? s.worktreeId : "",
    timestamp: typeof s.timestamp === "string" ? s.timestamp : new Date(0).toISOString(),
    dot: s.dot,
  };
  // Preserve the marker only when set, mirroring the backend's omitempty so
  // unmarked snapshots stay free of the field.
  if (s.sessionStart === true) {
    normalized.sessionStart = true;
  }
  return normalized;
}

function normalizeCommitSummary(commit: unknown): CommitSummary | null {
  if (!commit || typeof commit !== "object") {
    return null;
  }
  const c = commit as Record<string, unknown>;
  if (typeof c.hash !== "string" || c.hash === "") {
    return null;
  }
  return {
    hash: c.hash,
    shortHash: typeof c.shortHash === "string" ? c.shortHash : c.hash.slice(0, 7),
    subject: typeof c.subject === "string" ? c.subject : "",
    author: typeof c.author === "string" ? c.author : "",
    email: typeof c.email === "string" ? c.email : "",
    timestamp: typeof c.timestamp === "string" ? c.timestamp : new Date(0).toISOString(),
  };
}

function normalizeCollection(collection: unknown): Collection | null {
  if (!collection || typeof collection !== "object") {
    return null;
  }
  const c = collection as Record<string, unknown>;
  if (!Array.isArray(c.snapshots)) {
    return null;
  }

  const normalized: Collection = {
    id: Number.isFinite(c.id) ? (c.id as number) : 0,
    worktreeId: typeof c.worktreeId === "string" ? c.worktreeId : "",
    timestamp: typeof c.timestamp === "string" ? c.timestamp : new Date(0).toISOString(),
    snapshots: c.snapshots
      .map(normalizeSnapshot)
      .filter((snapshot): snapshot is Snapshot => snapshot !== null),
    commitHistory: Array.isArray(c.commitHistory)
      ? c.commitHistory.map(normalizeCommitSummary).filter((commit): commit is CommitSummary => commit !== null)
      : [],
  };
  if (typeof c.sessionId === "number" && c.sessionId > 0) {
    normalized.sessionId = c.sessionId;
  }
  return normalized;
}

function normalizePersistedSessionSummary(raw: unknown): PersistedSessionSummary | null {
  if (!raw || typeof raw !== "object") {
    return null;
  }
  const r = raw as Record<string, unknown>;
  if (!Number.isFinite(r.id) || typeof r.worktreeId !== "string" || r.worktreeId === "") {
    return null;
  }
  return {
    id: r.id as number,
    worktreeId: r.worktreeId,
    runId: Number.isFinite(r.runId) ? (r.runId as number) : 0,
    number: Number.isFinite(r.number) ? (r.number as number) : 0,
    createdAt: typeof r.createdAt === "string" ? r.createdAt : new Date(0).toISOString(),
    closedAt: typeof r.closedAt === "string" ? r.closedAt : null,
    closedReason: typeof r.closedReason === "string" ? r.closedReason : "",
    snapshotCount: Number.isFinite(r.snapshotCount) ? (r.snapshotCount as number) : 0,
    commitCount: Number.isFinite(r.commitCount) ? (r.commitCount as number) : 0,
  };
}

/**
 * Normalizes an untrusted GET /sessions JSON response.
 */
export function normalizePersistedSessionList(payload: unknown): PersistedSessionSummary[] {
  if (!Array.isArray(payload)) {
    return [];
  }
  return payload
    .map(normalizePersistedSessionSummary)
    .filter((summary): summary is PersistedSessionSummary => summary !== null);
}

function normalizePersistedSnapshot(raw: unknown): PersistedSnapshot | null {
  if (!raw || typeof raw !== "object") {
    return null;
  }
  const r = raw as Record<string, unknown>;
  if (typeof r.source !== "string") {
    return null;
  }
  return {
    position: Number.isFinite(r.position) ? (r.position as number) : 0,
    source: r.source,
    format: typeof r.format === "string" && r.format !== "" ? r.format : "dot",
    kind: typeof r.kind === "string" ? r.kind : "baseline",
    createdAt: typeof r.createdAt === "string" ? r.createdAt : new Date(0).toISOString(),
  };
}

function normalizePersistedCommit(raw: unknown): PersistedCommit | null {
  if (!raw || typeof raw !== "object") {
    return null;
  }
  const r = raw as Record<string, unknown>;
  if (typeof r.hash !== "string" || r.hash === "") {
    return null;
  }
  return {
    position: Number.isFinite(r.position) ? (r.position as number) : 0,
    hash: r.hash,
    subject: typeof r.subject === "string" ? r.subject : "",
  };
}

/**
 * Normalizes an untrusted GET /sessions/{id} JSON response.
 */
export function normalizePersistedSessionDetail(payload: unknown): PersistedSessionDetail | null {
  const summary = normalizePersistedSessionSummary(payload);
  if (!summary) {
    return null;
  }
  const r = payload as Record<string, unknown>;
  return {
    ...summary,
    snapshots: Array.isArray(r.snapshots)
      ? r.snapshots.map(normalizePersistedSnapshot).filter((s): s is PersistedSnapshot => s !== null)
      : [],
    commits: Array.isArray(r.commits)
      ? r.commits.map(normalizePersistedCommit).filter((c): c is PersistedCommit => c !== null)
      : [],
  };
}

/**
 * Normalizes untrusted SSE JSON payloads from the watch server.
 * Keeps only fields needed by the viewer state machine.
 */
export function normalizeGraphStreamPayload(payload: unknown): GraphStreamPayload {
  if (!payload || typeof payload !== "object") {
    return {
      worktrees: [],
      format: "dot",
      workingSnapshots: [],
      pastCollections: [],
    };
  }

  const p = payload as Record<string, unknown>;

  return {
    worktrees: Array.isArray(p.worktrees)
      ? p.worktrees.map(normalizeWorktree).filter((worktree): worktree is WorktreeDescriptor => worktree !== null)
      : [],
    workingSnapshots: Array.isArray(p.workingSnapshots)
      ? p.workingSnapshots.map(normalizeSnapshot).filter((snapshot): snapshot is Snapshot => snapshot !== null)
      : [],
    pastCollections: Array.isArray(p.pastCollections)
      ? p.pastCollections.map(normalizeCollection).filter((collection): collection is Collection => collection !== null)
      : [],
    latestWorkingId: Number.isFinite(p.latestWorkingId) ? (p.latestWorkingId as number) : 0,
    latestPastCollectionId: Number.isFinite(p.latestPastCollectionId) ? (p.latestPastCollectionId as number) : 0,
    format: typeof p.format === "string" && p.format !== "" ? p.format : "dot",
  };
}
