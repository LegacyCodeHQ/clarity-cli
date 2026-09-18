/**
 * State management for the watch viewer.
 * Pure functions for state transitions and view model computation.
 */

import type {
  Snapshot,
  Collection,
  GraphStreamPayload,
  WorktreeDescriptor,
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

  // Session-global render format ("dot" or "mermaid") from the latest payload.
  format: string;
}

export interface SourceOption {
  value: string;
  text: string;
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
  });
}

export function applySliderInput(state: ViewerState, rawValue: string): ViewerState {
  const next = normalizeState(state);
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
    });
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

  return [...liveOptions, ...frozenOptions, ...collectionOptions];
}

export function getViewModel(state: ViewerState, timeFormatter: TimeFormatter = formatTime): ViewModel {
  const normalized = normalizeState(state);
  const allowsLive = selectedWorktreeAllowsLive(normalized);
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
