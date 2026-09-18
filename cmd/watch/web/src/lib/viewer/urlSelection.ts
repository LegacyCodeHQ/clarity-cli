/**
 * Persists the viewer's current selection (repo tab, archived session,
 * scrub position) to and from the page's URL query string, so a browser
 * refresh restores the view instead of always landing back on Live.
 *
 * Pure string in/out — no `window`/`history` access here — so this stays
 * testable under vitest's default node environment. The browser wiring
 * (reading `location.search`, calling `history.replaceState`) lives in
 * graphStore.ts.
 */

import { DEFAULT_REPO_ID, type ViewerState } from './viewerState';

export interface URLSelection {
  repo?: string;
  collectionID?: number;
  snapshotIndex?: number;
}

export function parseSelectionFromSearch(search: string): URLSelection {
  const params = new URLSearchParams(search);
  const result: URLSelection = {};

  const repo = params.get('repo');
  if (repo) {
    result.repo = repo;
  }

  const sessionRaw = params.get('session');
  if (sessionRaw !== null) {
    const id = Number(sessionRaw);
    if (Number.isFinite(id)) {
      result.collectionID = id;
    }
  }

  const snapshotRaw = params.get('snapshot');
  if (snapshotRaw !== null) {
    const idx = Number(snapshotRaw);
    if (Number.isFinite(idx) && idx >= 0) {
      result.snapshotIndex = idx;
    }
  }

  return result;
}

export function buildSearchFromState(
  state: Pick<ViewerState, 'selectedRepoID' | 'selectedCollectionID' | 'selectedCollectionSnapshotIndex' | 'liveSnapshotIndex'>,
): string {
  const params = new URLSearchParams();

  if (state.selectedRepoID && state.selectedRepoID !== DEFAULT_REPO_ID) {
    params.set('repo', state.selectedRepoID);
  }

  if (state.selectedCollectionID !== null) {
    params.set('session', String(state.selectedCollectionID));
    params.set('snapshot', String(state.selectedCollectionSnapshotIndex));
  } else if (state.liveSnapshotIndex !== null) {
    params.set('snapshot', String(state.liveSnapshotIndex));
  }

  const serialized = params.toString();
  return serialized ? `?${serialized}` : '';
}
