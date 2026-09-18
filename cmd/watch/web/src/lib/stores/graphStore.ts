/**
 * Svelte store wrapper for viewer state management.
 * Provides reactive state updates and view model derivation.
 */

import { writable, derived } from 'svelte/store';
import {
  normalizeState,
  mergePayload,
  applySliderInput,
  applyTimelineStep,
  applyLiveSelection,
  applySourceSelection,
  selectWorktree,
  setPersistedSessions,
  applyPersistedSessionDetail,
  getViewModel,
  DEFAULT_WORKTREE_ID,
  type ViewerState,
  type ViewModel,
} from '../viewer/viewerState';
import { parseSelectionFromSearch, buildSearchFromState, type URLSelection } from '../viewer/urlSelection';
import {
  normalizePersistedSessionDetail,
  type GraphStreamPayload,
  type PersistedSessionSummary,
} from '../protocol/viewerProtocol';

// Fetches one persisted session's full content on demand (CLR-98/99) —
// never called eagerly, only once the user selects a `session:<id>`
// option. Returns null on any failure (network error, non-2xx, malformed
// body); the caller treats that the same as "couldn't load."
async function fetchPersistedSessionDetail(id: number) {
  try {
    const res = await fetch(`/sessions/${id}`);
    if (!res.ok) {
      return null;
    }
    return normalizePersistedSessionDetail(await res.json());
  } catch (err) {
    console.error('Failed to fetch session detail:', err);
    return null;
  }
}

function createGraphStore() {
  // Read once at startup: the selection a refresh (or a shared/bookmarked
  // link) should restore. It can't be validated against real data yet —
  // worktrees/collections don't exist until the first SSE payload arrives —
  // so it's applied on that first mergePayload call, not baked into
  // initialState, and normalizeState's usual clamping falls back to Live if
  // it turns out to reference a session/snapshot that no longer exists.
  let pendingSelection: URLSelection | null =
    typeof window !== 'undefined' ? parseSelectionFromSearch(window.location.search) : null;
  if (pendingSelection && Object.keys(pendingSelection).length === 0) {
    pendingSelection = null;
  }

  const initialState: ViewerState = normalizeState({
    worktrees: [],
    selectedWorktreeID: pendingSelection?.worktree ?? DEFAULT_WORKTREE_ID,
    byWorktree: {},
    workingSnapshots: [],
    pastCollections: [],
    selectedCollectionID: null,
    selectedCollectionSnapshotIndex: 0,
    liveSnapshotIndex: null,
  });

  const { subscribe, update } = writable<ViewerState>(initialState);

  // Reflects the current selection into the URL (via replaceState, so
  // scrubbing/switching sessions doesn't spam the back-button history) each
  // time it changes, so a later refresh can restore it.
  function syncURL(state: ViewerState) {
    if (typeof window === 'undefined' || typeof history === 'undefined') {
      return;
    }
    const search = buildSearchFromState(state);
    if (search === window.location.search) {
      return;
    }
    history.replaceState(history.state, '', `${window.location.pathname}${search}${window.location.hash}`);
  }

  function applyAndSync(fn: (state: ViewerState) => ViewerState) {
    update((state) => {
      const next = fn(state);
      syncURL(next);
      return next;
    });
  }

  return {
    subscribe,

    mergePayload: (payload: GraphStreamPayload) => {
      applyAndSync((state) => {
        if (!pendingSelection) {
          return mergePayload(state, payload);
        }
        const seeded = mergePayload(
          {
            ...state,
            selectedWorktreeID: pendingSelection.worktree ?? state.selectedWorktreeID,
            selectedCollectionID: pendingSelection.collectionID ?? null,
            selectedCollectionSnapshotIndex: pendingSelection.snapshotIndex ?? 0,
            liveSnapshotIndex: pendingSelection.collectionID === undefined
              ? pendingSelection.snapshotIndex ?? null
              : null,
          },
          payload,
        );
        pendingSelection = null;
        return seeded;
      });
    },

    onSliderInput: (rawValue: string) => {
      applyAndSync(state => applySliderInput(state, rawValue));
    },

    onTimelineStep: (delta: number) => {
      applyAndSync(state => applyTimelineStep(state, delta));
    },

    onJumpToLatest: () => {
      applyAndSync(state => applyLiveSelection(state));
    },

    onSourceChange: (selected: string) => {
      // applySourceSelection's `session:<id>` branch only does the
      // optimistic half (marks it selected + loading) — it can't fetch
      // itself and stay pure. Kick off the fetch here and feed the result
      // back through applyPersistedSessionDetail once it resolves.
      applyAndSync(state => applySourceSelection(state, selected));
      if (selected.startsWith('session:')) {
        const id = Number(selected.split(':')[1]);
        if (Number.isFinite(id)) {
          fetchPersistedSessionDetail(id).then((detail) => {
            applyAndSync(state => applyPersistedSessionDetail(state, id, detail));
          });
        }
      }
    },

    onSelectWorktree: (worktreeID: string) => {
      applyAndSync(state => selectWorktree(state, worktreeID));
    },

    // Replaces the eager, metadata-only persisted-session listing (CLR-98's
    // GET /sessions), fetched once on attach — see App.svelte. Doesn't
    // touch the URL: the listing alone never changes what's selected/
    // rendered, only what getSourceOptions offers.
    setPersistedSessions: (sessions: PersistedSessionSummary[]) => {
      update(state => setPersistedSessions(state, sessions));
    },

    reset: () => {
      update(() => initialState);
    },
  };
}

export const graphStore = createGraphStore();

/**
 * Derived store that computes the view model from the current state
 */
export const viewModel = derived<typeof graphStore, ViewModel>(
  graphStore,
  ($state) => getViewModel($state)
);
