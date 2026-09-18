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
  getViewModel,
  DEFAULT_WORKTREE_ID,
  type ViewerState,
  type ViewModel,
} from '../viewer/viewerState';
import { parseSelectionFromSearch, buildSearchFromState, type URLSelection } from '../viewer/urlSelection';
import type { GraphStreamPayload } from '../protocol/viewerProtocol';

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
      applyAndSync(state => applySourceSelection(state, selected));
    },

    onSelectWorktree: (worktreeID: string) => {
      applyAndSync(state => selectWorktree(state, worktreeID));
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
