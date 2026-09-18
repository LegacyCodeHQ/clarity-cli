/**
 * State management for the watch viewer.
 * Pure functions for state transitions and view model computation.
 */

import type {
  Snapshot,
  Collection,
  GraphStreamPayload,
  WorktreeDescriptor,
  PersistedSessionSummary,
  PersistedSessionDetail,
} from '../protocol/viewerProtocol';

/**
 * Snapshots and archived cycles for a single working tree (tab).
 */
export interface WorktreeBucket {
  working: Snapshot[];
  past: Collection[];
}

export interface ViewerState {
  // Multi-worktree: the registered tab set + which tab is active.
  worktrees: WorktreeDescriptor[];
  selectedWorktreeID: string;
  byWorktree: Record<string, WorktreeBucket>;

  // Effective view for the selected worktree — derived from
  // byWorktree[selectedWorktreeID] on every state update so downstream
  // timeline/graph code stays unchanged.
  workingSnapshots: Snapshot[];
  pastCollections: Collection[];

  selectedCollectionID: number | null;
  selectedCollectionSnapshotIndex: number;
  liveSnapshotIndex: number | null;

  // Persisted session history (CLR-98/99): a project-wide, metadata-only
  // listing fetched once on attach (GET /sessions) — every worktree, every
  // past `clarity watch` run. Selecting one of these is a third,
  // mutually-exclusive mode alongside the live working set and an
  // in-memory collection; see selectedPersistedSessionID.
  persistedSessions: PersistedSessionSummary[];
  // Non-null exactly when a past-run session is the active selection.
  // Mutually exclusive with selectedCollectionID (selecting one clears the
  // other — see applySourceSelection/selectWorktree/applyLiveSelection).
  selectedPersistedSessionID: number | null;
  // The fetched detail (snapshots + commits) for selectedPersistedSessionID,
  // null while loading or when nothing is selected — GET /sessions/{id} is
  // only called on demand, never eagerly, so this starts empty even once
  // selectedPersistedSessionID is set (see persistedSessionLoading).
  persistedSessionDetail: PersistedSessionDetail | null;
  persistedSessionLoading: boolean;
  persistedSessionSnapshotIndex: number;

  // Session-global render format ("dot" or "mermaid") from the latest payload.
  format: string;
}

export interface SourceOption {
  value: string;
  text: string;
  // Options sharing the same group render under one <optgroup> (see
  // SourceSelector.svelte) — used to cluster a past run's sessions
  // together. Consecutive options must share a group for this to render
  // correctly; getSourceOptions guarantees that ordering. Absent for the
  // live/frozen/in-memory-collection options, which stay ungrouped exactly
  // as before this field existed.
  group?: string;
}

export interface TimelineViewModel {
  modeText: string;
  sliderDisabled: boolean;
  sliderMax: string;
  sliderValue: string;
  liveButtonDisabled: boolean;
  metaText: string;
  // Index of the session-start snapshot within the currently displayed list
  // (live working set or selected collection), or null if it isn't present —
  // e.g. after a commit archives the cycle the marker belonged to.
  sessionStartIndex: number | null;
}

export interface ViewModel {
  state: ViewerState;
  sourceValue: string;
  sourceOptions: SourceOption[];
  renderDot: string | null;
  renderFormat: string;
  timeline: TimelineViewModel;
}

type TimeFormatter = (timestamp: string) => string;

export const DEFAULT_WORKTREE_ID = "main";

function clamp(value: number, min: number, max: number): number {
  return Math.max(min, Math.min(value, max));
}

function findSessionStartIndex(snapshots: Snapshot[]): number | null {
  const index = snapshots.findIndex((snapshot) => snapshot.sessionStart === true);
  return index === -1 ? null : index;
}

function formatTime(timestamp: string): string {
  return new Date(timestamp).toLocaleTimeString();
}

export function formatSnapshotMeta(
  snapshot: Snapshot,
  index: number,
  total: number,
  timeFormatter: TimeFormatter = formatTime
): string {
  return `#${index + 1}/${total} | id ${snapshot.id} | ${timeFormatter(snapshot.timestamp)}`;
}

export function getSelectedCollection(state: ViewerState): Collection | null {
  if (state.selectedCollectionID === null) {
    return null;
  }
  return state.pastCollections.find((collection) => collection.id === state.selectedCollectionID) || null;
}

function selectedWorktreeAllowsLive(state: Pick<ViewerState, "worktrees" | "selectedWorktreeID">): boolean {
  const worktree = state.worktrees.find((w) => w.id === state.selectedWorktreeID);
  return worktree ? worktree.active !== false : true;
}

/**
 * Picks the next selected worktree when the previous selection becomes
 * invalid (e.g., its tab was removed). Prefers the existing selection, then
 * "main", then the first worktree in the list.
 */
function resolveSelectedWorktreeID(worktrees: WorktreeDescriptor[], current: string): string {
  if (worktrees.length === 0) {
    return current || DEFAULT_WORKTREE_ID;
  }
  if (worktrees.some((w) => w.id === current)) {
    return current;
  }
  const primary = worktrees.find((w) => w.id === DEFAULT_WORKTREE_ID);
  if (primary) {
    return primary.id;
  }
  return worktrees[0].id;
}

function projectBucketsForState(
  state: ViewerState,
  selectedWorktreeID: string,
): { workingSnapshots: Snapshot[]; pastCollections: Collection[] } {
  const bucket = state.byWorktree[selectedWorktreeID];
  return {
    workingSnapshots: bucket ? bucket.working : [],
    pastCollections: bucket ? bucket.past : [],
  };
}

export function normalizeState(state: Partial<ViewerState>): ViewerState {
  const worktrees = Array.isArray(state.worktrees) ? state.worktrees : [];
  const selectedWorktreeID = resolveSelectedWorktreeID(worktrees, state.selectedWorktreeID ?? DEFAULT_WORKTREE_ID);

  // Ad-hoc callers (notably test fixtures) can set workingSnapshots /
  // pastCollections directly without populating a byWorktree bucket. When the
  // selected worktree has no bucket yet, treat those arrays as its initial
  // state. mergePayload clears these arrays explicitly so an empty payload
  // doesn't bring stale snapshots back through this fallback.
  const fallbackWorking = Array.isArray(state.workingSnapshots) ? state.workingSnapshots : [];
  const fallbackPast = Array.isArray(state.pastCollections) ? state.pastCollections : [];
  const byWorktree: Record<string, WorktreeBucket> = state.byWorktree ? { ...state.byWorktree } : {};
  if (!byWorktree[selectedWorktreeID] && (fallbackWorking.length > 0 || fallbackPast.length > 0)) {
    byWorktree[selectedWorktreeID] = { working: fallbackWorking, past: fallbackPast };
  }

  const projected = projectBucketsForState({ ...(state as ViewerState), byWorktree }, selectedWorktreeID);

  const next: ViewerState = {
    worktrees,
    selectedWorktreeID,
    byWorktree,
    workingSnapshots: projected.workingSnapshots,
    pastCollections: projected.pastCollections,
    selectedCollectionID: state.selectedCollectionID ?? null,
    selectedCollectionSnapshotIndex: Number.isFinite(state.selectedCollectionSnapshotIndex)
      ? state.selectedCollectionSnapshotIndex!
      : 0,
    liveSnapshotIndex: state.liveSnapshotIndex === null || Number.isFinite(state.liveSnapshotIndex)
      ? state.liveSnapshotIndex ?? null
      : null,
    persistedSessions: Array.isArray(state.persistedSessions) ? state.persistedSessions : [],
    selectedPersistedSessionID: state.selectedPersistedSessionID ?? null,
    persistedSessionDetail: state.persistedSessionDetail ?? null,
    persistedSessionLoading: state.persistedSessionLoading ?? false,
    persistedSessionSnapshotIndex: Number.isFinite(state.persistedSessionSnapshotIndex)
      ? state.persistedSessionSnapshotIndex!
      : 0,
    format: state.format ?? "dot",
  };

  if (next.workingSnapshots.length === 0) {
    next.liveSnapshotIndex = null;
  }

  const selectedCollection = getSelectedCollection(next);
  if (next.selectedCollectionID !== null && !selectedCollection) {
    next.selectedCollectionID = null;
    next.selectedCollectionSnapshotIndex = 0;
  }

  if (next.selectedPersistedSessionID !== null) {
    const snapshots = next.persistedSessionDetail?.snapshots ?? [];
    next.persistedSessionSnapshotIndex = snapshots.length > 0
      ? clamp(next.persistedSessionSnapshotIndex, 0, snapshots.length - 1)
      : 0;
  }

  if (
    next.selectedCollectionID === null
    && !selectedWorktreeAllowsLive(next)
    && next.workingSnapshots.length === 0
    && next.pastCollections.length > 0
  ) {
    const latest = next.pastCollections[next.pastCollections.length - 1]!;
    next.selectedCollectionID = latest.id;
    // Land on the most recent snapshot of the session, not the first — the
    // removed worktree's final state is what the user expects to see.
    const latestSnapshots = latest.snapshots || [];
    next.selectedCollectionSnapshotIndex = latestSnapshots.length > 0 ? latestSnapshots.length - 1 : 0;
  }

  if (next.selectedCollectionID === null) {
    const total = next.workingSnapshots.length;
    const latestIndex = total > 0 ? total - 1 : 0;
    if (next.liveSnapshotIndex !== null) {
      next.liveSnapshotIndex = clamp(next.liveSnapshotIndex, 0, latestIndex);
      if (next.liveSnapshotIndex === latestIndex) {
        next.liveSnapshotIndex = null;
      }
    }
    return next;
  }

  const collection = getSelectedCollection(next);
  const snapshots = collection ? collection.snapshots || [] : [];
  if (snapshots.length === 0) {
    next.selectedCollectionSnapshotIndex = 0;
    return next;
  }
  next.selectedCollectionSnapshotIndex = clamp(
    next.selectedCollectionSnapshotIndex,
    0,
    snapshots.length - 1,
  );
  return next;
}

/**
 * Buckets a flat payload by worktreeId. Snapshots/collections without a
 * worktreeId fall into the main bucket — keeps backward-tolerance with
 * older payloads and with single-worktree callers.
 */
function bucketPayload(payload: GraphStreamPayload): Record<string, WorktreeBucket> {
  const byWorktree: Record<string, WorktreeBucket> = {};
  const knownIds = new Set<string>();
  for (const worktree of payload.worktrees || []) {
    knownIds.add(worktree.id);
    byWorktree[worktree.id] = { working: [], past: [] };
  }
  for (const snap of payload.workingSnapshots || []) {
    const id = snap.worktreeId || DEFAULT_WORKTREE_ID;
    if (!byWorktree[id]) {
      byWorktree[id] = { working: [], past: [] };
    }
    byWorktree[id].working.push(snap);
  }
  for (const coll of payload.pastCollections || []) {
    const id = coll.worktreeId || DEFAULT_WORKTREE_ID;
    if (!byWorktree[id]) {
      byWorktree[id] = { working: [], past: [] };
    }
    byWorktree[id].past.push(coll);
  }
  // Drop any synthesized empty buckets that weren't declared by worktrees[]
  // AND received no snapshots — keeps `byWorktree` honest.
  for (const id of Object.keys(byWorktree)) {
    if (!knownIds.has(id) && byWorktree[id].working.length === 0 && byWorktree[id].past.length === 0) {
      delete byWorktree[id];
    }
  }
  return byWorktree;
}

export function mergePayload(state: ViewerState, payload: GraphStreamPayload): ViewerState {
  const worktrees = payload.worktrees || state.worktrees;
  const byWorktree = bucketPayload(payload);
  return normalizeState({
    ...state,
    worktrees,
    byWorktree,
    format: payload.format ?? state.format ?? "dot",
    // Discard the previous projection so it can't leak through normalizeState's
    // fallback when the new bucket is empty for the selected worktree.
    workingSnapshots: [],
    pastCollections: [],
  });
}

export function selectWorktree(state: ViewerState, worktreeID: string): ViewerState {
  if (!state.worktrees.some((w) => w.id === worktreeID)) {
    return state;
  }
  if (state.selectedWorktreeID === worktreeID) {
    return state;
  }
  // Switching tabs resets the per-tab timeline selection.
  return normalizeState({
    ...state,
    selectedWorktreeID: worktreeID,
    selectedCollectionID: null,
    selectedCollectionSnapshotIndex: 0,
    liveSnapshotIndex: null,
    selectedPersistedSessionID: null,
    persistedSessionDetail: null,
    persistedSessionLoading: false,
  });
}

/**
 * Replaces the eager, metadata-only persisted-session listing (GET
 * /sessions). Doesn't touch any current selection — the listing is only
 * ever consulted by getSourceOptions, never used to drive rendering
 * directly (see selectPersistedSession/applyPersistedSessionDetail for
 * that).
 */
export function setPersistedSessions(state: ViewerState, sessions: PersistedSessionSummary[]): ViewerState {
  return normalizeState({ ...state, persistedSessions: sessions });
}

/**
 * The optimistic half of selecting a past-run session: marks it as
 * selected and loading, clearing any other selection, before the caller
 * (graphStore, which owns the actual fetch — this module stays pure) has
 * resolved GET /sessions/{id}. See applyPersistedSessionDetail for the
 * other half.
 */
export function selectPersistedSession(state: ViewerState, id: number): ViewerState {
  return normalizeState({
    ...state,
    selectedCollectionID: null,
    selectedCollectionSnapshotIndex: 0,
    liveSnapshotIndex: null,
    selectedPersistedSessionID: id,
    persistedSessionDetail: null,
    persistedSessionLoading: true,
    persistedSessionSnapshotIndex: 0,
  });
}

/**
 * Applies a resolved GET /sessions/{id} fetch. Ignored if the selection has
 * already moved on to something else by the time the fetch resolves (a
 * stale response arriving after the user picked a different source) — id
 * must still match selectedPersistedSessionID. detail is null on a failed
 * fetch, which still clears the loading flag so the UI doesn't spin
 * forever.
 */
export function applyPersistedSessionDetail(
  state: ViewerState,
  id: number,
  detail: PersistedSessionDetail | null,
): ViewerState {
  if (state.selectedPersistedSessionID !== id) {
    return state;
  }
  const snapshots = detail?.snapshots ?? [];
  return normalizeState({
    ...state,
    persistedSessionDetail: detail,
    persistedSessionLoading: false,
    persistedSessionSnapshotIndex: snapshots.length > 0 ? snapshots.length - 1 : 0,
  });
}

export function applySliderInput(state: ViewerState, rawValue: string): ViewerState {
  const next = normalizeState(state);

  if (next.selectedPersistedSessionID !== null) {
    const snapshots = next.persistedSessionDetail?.snapshots ?? [];
    if (snapshots.length === 0) {
      return next;
    }
    next.persistedSessionSnapshotIndex = clamp(Number(rawValue || "0"), 0, snapshots.length - 1);
    return next;
  }

  if (next.selectedCollectionID === null) {
    if (next.workingSnapshots.length === 0) {
      return next;
    }
    const latestIndex = next.workingSnapshots.length - 1;
    const idx = clamp(Number(rawValue || "0"), 0, latestIndex);
    next.liveSnapshotIndex = idx === latestIndex ? null : idx;
    return normalizeState(next);
  }

  const collection = getSelectedCollection(next);
  const snapshots = collection ? collection.snapshots || [] : [];
  if (snapshots.length === 0) {
    return next;
  }
  next.selectedCollectionSnapshotIndex = clamp(Number(rawValue || "0"), 0, snapshots.length - 1);
  return normalizeState(next);
}

export function applyTimelineStep(state: ViewerState, delta: number): ViewerState {
  const next = normalizeState(state);
  if (!Number.isFinite(delta) || delta === 0) {
    return next;
  }

  if (next.selectedPersistedSessionID !== null) {
    const total = next.persistedSessionDetail?.snapshots.length ?? 0;
    if (total <= 1) {
      return next;
    }
    return applySliderInput(next, String(next.persistedSessionSnapshotIndex + delta));
  }

  if (next.selectedCollectionID === null) {
    const total = next.workingSnapshots.length;
    if (total <= 1) {
      return next;
    }
    const currentIndex = next.liveSnapshotIndex === null ? total - 1 : next.liveSnapshotIndex;
    return applySliderInput(next, String(currentIndex + delta));
  }

  const collection = getSelectedCollection(next);
  const snapshots = collection ? collection.snapshots || [] : [];
  if (snapshots.length <= 1) {
    return next;
  }
  return applySliderInput(next, String(next.selectedCollectionSnapshotIndex + delta));
}

export function applyLiveSelection(state: ViewerState): ViewerState {
  return normalizeState({
    ...state,
    liveSnapshotIndex: null,
    selectedCollectionID: null,
    selectedCollectionSnapshotIndex: 0,
    selectedPersistedSessionID: null,
    persistedSessionDetail: null,
    persistedSessionLoading: false,
  });
}

export function applySourceSelection(state: ViewerState, selected: string): ViewerState {
  if (selected === "live") {
    return applyLiveSelection(state);
  }
  if (selected === "frozen") {
    return normalizeState({
      ...state,
      liveSnapshotIndex: null,
      selectedCollectionID: null,
      selectedCollectionSnapshotIndex: 0,
      selectedPersistedSessionID: null,
      persistedSessionDetail: null,
      persistedSessionLoading: false,
    });
  }
  if (selected.startsWith("session:")) {
    const id = Number(selected.split(":")[1]);
    if (!Number.isFinite(id)) {
      return applyLiveSelection(state);
    }
    // Optimistic only — the caller (graphStore) owns fetching GET
    // /sessions/{id} and feeding the result back through
    // applyPersistedSessionDetail once it resolves.
    return selectPersistedSession(state, id);
  }
  if (!selected.startsWith("collection:")) {
    return applyLiveSelection(state);
  }

  const selectedID = Number(selected.split(":")[1]);
  if (!Number.isFinite(selectedID)) {
    return applyLiveSelection(state);
  }

  // Land on the most recent snapshot of the chosen session, consistent with
  // live mode (which shows the newest working snapshot) and the worktree-
  // removal auto-select. The slider still lets the user scrub back to the start.
  const target = state.pastCollections.find((c) => c.id === selectedID);
  const snapshots = target ? target.snapshots || [] : [];
  return normalizeState({
    ...state,
    selectedCollectionID: selectedID,
    selectedCollectionSnapshotIndex: snapshots.length > 0 ? snapshots.length - 1 : 0,
    selectedPersistedSessionID: null,
    persistedSessionDetail: null,
    persistedSessionLoading: false,
  });
}

export function getSourceOptions(state: ViewerState, timeFormatter: TimeFormatter = formatTime): SourceOption[] {
  const allowsLive = selectedWorktreeAllowsLive(state);
  const liveOptions: SourceOption[] = allowsLive
    ? [{
      value: "live",
      text: "Current working directory (live)",
    }]
    : [];
  const frozenOptions: SourceOption[] = !allowsLive && state.workingSnapshots.length > 0
    ? [{
      value: "frozen",
      text: "Removed working directory snapshot",
    }]
    : [];
  const orderedCollections = [...state.pastCollections].reverse();
  const collectionOptions = orderedCollections.map((collection, index) => {
    const number = state.pastCollections.length - index;
    const snapshots = collection.snapshots || [];
    const commitHistory = collection.commitHistory || [];
    // A session can bundle several commits landed between polls; show the
    // most recent one, matching what a user thinks of as "what HEAD is now".
    const latestCommit = commitHistory[commitHistory.length - 1];
    const label = latestCommit
      ? latestCommit.subject
      : `(${snapshots.length} snapshots, ${timeFormatter(collection.timestamp)})`;
    return {
      value: `collection:${collection.id}`,
      text: `#${number} ${label}`,
    };
  });

  return [...liveOptions, ...frozenOptions, ...collectionOptions, ...getPersistedSessionOptions(state, timeFormatter)];
}

/**
 * The past-runs block of the dropdown (CLR-99): every persisted session
 * for the selected worktree, excluding the current run's own still-open
 * session (that's the live working set, not history) and excluding
 * anything already visible via pastCollections (see
 * protocol.SnapshotCollection.SessionID / Collection.sessionId) — the
 * current run's own closed sessions must show up once, not twice.
 *
 * Sorted by run (newest first) then session number (newest first) within
 * a run, so consecutive options share a `group` label — required for
 * SourceSelector.svelte's <optgroup> rendering to group correctly.
 */
function getPersistedSessionOptions(state: ViewerState, timeFormatter: TimeFormatter): SourceOption[] {
  const alreadyShown = new Set(
    state.pastCollections
      .map((c) => c.sessionId)
      .filter((id): id is number => typeof id === "number" && id > 0),
  );

  const sessions = state.persistedSessions
    .filter((s) => s.worktreeId === state.selectedWorktreeID)
    .filter((s) => s.closedAt !== null)
    .filter((s) => !alreadyShown.has(s.id))
    .sort((a, b) => b.runId - a.runId || b.number - a.number);

  // A run's own started_at isn't in the summary (only its id) — the
  // earliest session recorded under a run is a reasonable, honest stand-in
  // for when that run's visible history begins.
  const runStartTimes = new Map<number, string>();
  for (const s of sessions) {
    const existing = runStartTimes.get(s.runId);
    if (!existing || s.createdAt < existing) {
      runStartTimes.set(s.runId, s.createdAt);
    }
  }

  return sessions.map((s) => {
    const detail = s.closedReason === "committed"
      ? `${s.commitCount} commit${s.commitCount === 1 ? "" : "s"}`
      : s.closedReason || "closed";
    return {
      value: `session:${s.id}`,
      text: `#${s.number} (${detail}, ${timeFormatter(s.createdAt)})`,
      group: `Run started ${timeFormatter(runStartTimes.get(s.runId)!)}`,
    };
  });
}

export interface SourceOptionBlock {
  // null for the ungrouped live/frozen/in-memory-collection options —
  // rendered as plain top-level <option>s rather than inside an
  // <optgroup> (see SourceSelector.svelte).
  group: string | null;
  options: SourceOption[];
}

/**
 * Groups consecutive SourceOptions sharing the same `group` label into
 * blocks, for SourceSelector.svelte to render as <optgroup>s. Options must
 * already be in group-consecutive order — getSourceOptions guarantees
 * this — since HTML <optgroup> can't represent a group split across two
 * non-adjacent ranges.
 */
export function groupSourceOptions(options: SourceOption[]): SourceOptionBlock[] {
  const blocks: SourceOptionBlock[] = [];
  for (const option of options) {
    const group = option.group ?? null;
    const last = blocks[blocks.length - 1];
    if (last && last.group === group) {
      last.options.push(option);
    } else {
      blocks.push({ group, options: [option] });
    }
  }
  return blocks;
}

export function getViewModel(state: ViewerState, timeFormatter: TimeFormatter = formatTime): ViewModel {
  const normalized = normalizeState(state);
  const allowsLive = selectedWorktreeAllowsLive(normalized);

  if (normalized.selectedPersistedSessionID !== null) {
    return getPersistedSessionViewModel(normalized, allowsLive, timeFormatter);
  }

  const sourceValue = normalized.selectedCollectionID === null
    ? allowsLive
      ? "live"
      : normalized.workingSnapshots.length > 0
        ? "frozen"
        : ""
    : `collection:${normalized.selectedCollectionID}`;

  if (normalized.selectedCollectionID === null) {
    const total = normalized.workingSnapshots.length;
    const latestIndex = total > 0 ? total - 1 : 0;
    const selectedIndex = normalized.liveSnapshotIndex === null
      ? latestIndex
      : normalized.liveSnapshotIndex;

    return {
      state: normalized,
      sourceValue,
      sourceOptions: getSourceOptions(normalized, timeFormatter),
      renderDot: total > 0 ? normalized.workingSnapshots[selectedIndex]!.dot : null,
      renderFormat: normalized.format,
      timeline: {
        modeText: !allowsLive
          ? "Removed working directory snapshot"
          : normalized.liveSnapshotIndex === null
          ? "Working directory (live)"
          : "Working directory snapshot",
        sliderDisabled: total <= 1,
        sliderMax: total > 0 ? String(total - 1) : "0",
        sliderValue: total > 0 ? String(selectedIndex) : "0",
        liveButtonDisabled: !allowsLive || total === 0 || normalized.liveSnapshotIndex === null,
        metaText: total === 0
          ? "0 working snapshots"
          : `${total} ${allowsLive ? "working" : "frozen working"} snapshots | ${formatSnapshotMeta(
            normalized.workingSnapshots[selectedIndex]!,
            selectedIndex,
            total,
            timeFormatter,
          )}`,
        sessionStartIndex: findSessionStartIndex(normalized.workingSnapshots),
      },
    };
  }

  const selectedCollection = getSelectedCollection(normalized);
  const snapshots = selectedCollection ? selectedCollection.snapshots || [] : [];
  const total = snapshots.length;

  return {
    state: normalized,
    sourceValue,
    sourceOptions: getSourceOptions(normalized, timeFormatter),
    renderDot: total > 0 ? snapshots[normalized.selectedCollectionSnapshotIndex]!.dot : null,
    renderFormat: normalized.format,
    timeline: {
      modeText: "Session snapshots",
      sliderDisabled: total <= 1,
      sliderMax: total > 0 ? String(total - 1) : "0",
      sliderValue: total > 0 ? String(normalized.selectedCollectionSnapshotIndex) : "0",
      liveButtonDisabled: !allowsLive,
      metaText: total === 0
        ? "Session is empty"
        : `${total} snapshots | ${formatSnapshotMeta(
          snapshots[normalized.selectedCollectionSnapshotIndex]!,
          normalized.selectedCollectionSnapshotIndex,
          total,
          timeFormatter,
        )}`,
      sessionStartIndex: findSessionStartIndex(snapshots),
    },
  };
}

/**
 * The view model for a selected past-run session (CLR-99) — a third mode
 * alongside the live working set and an in-memory collection. Renders a
 * loading placeholder while GET /sessions/{id} is in flight
 * (persistedSessionLoading), since unlike the other two modes this one's
 * content isn't available synchronously from state alone.
 */
function getPersistedSessionViewModel(
  state: ViewerState,
  allowsLive: boolean,
  timeFormatter: TimeFormatter,
): ViewModel {
  const sourceValue = `session:${state.selectedPersistedSessionID}`;
  const sourceOptions = getSourceOptions(state, timeFormatter);

  if (state.persistedSessionLoading || !state.persistedSessionDetail) {
    return {
      state,
      sourceValue,
      sourceOptions,
      renderDot: null,
      renderFormat: state.format,
      timeline: {
        modeText: "Loading session…",
        sliderDisabled: true,
        sliderMax: "0",
        sliderValue: "0",
        liveButtonDisabled: !allowsLive,
        metaText: state.persistedSessionLoading ? "Loading…" : "Failed to load session",
        sessionStartIndex: null,
      },
    };
  }

  const detail = state.persistedSessionDetail;
  const snapshots = detail.snapshots;
  const total = snapshots.length;
  const index = state.persistedSessionSnapshotIndex;
  const current = total > 0 ? snapshots[index]! : null;

  return {
    state,
    sourceValue,
    sourceOptions,
    renderDot: current?.source ?? null,
    // Every persisted snapshot in a session was captured under the same
    // `clarity watch --format` flag, so the first snapshot's format
    // represents the whole session; fall back to the live session's own
    // format for an empty session.
    renderFormat: snapshots[0]?.format || state.format,
    timeline: {
      modeText: `Persisted session #${detail.number}`,
      sliderDisabled: total <= 1,
      sliderMax: total > 0 ? String(total - 1) : "0",
      sliderValue: total > 0 ? String(index) : "0",
      liveButtonDisabled: !allowsLive,
      metaText: total === 0
        ? "Session is empty"
        : `${total} snapshots | #${index + 1}/${total} | ${timeFormatter(current!.createdAt)}`,
      // Persisted snapshots don't carry the in-process sessionStart marker
      // (that's a live-payload-only concept — see Snapshot.sessionStart).
      sessionStartIndex: null,
    },
  };
}
