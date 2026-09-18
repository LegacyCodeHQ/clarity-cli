import { describe, it, expect } from 'vitest';
import { parseSelectionFromSearch, buildSearchFromState } from './urlSelection';

describe('parseSelectionFromSearch', () => {
  it('returns an empty selection for no query string', () => {
    expect(parseSelectionFromSearch('')).toEqual({});
  });

  it('parses repo, session, and snapshot together', () => {
    expect(parseSelectionFromSearch('?repo=wt-abc&session=3&snapshot=2')).toEqual({
      repo: 'wt-abc',
      collectionID: 3,
      snapshotIndex: 2,
    });
  });

  it('parses a live scrub position with no session', () => {
    expect(parseSelectionFromSearch('?snapshot=1')).toEqual({ snapshotIndex: 1 });
  });

  it('ignores a non-numeric session id', () => {
    expect(parseSelectionFromSearch('?session=abc')).toEqual({});
  });

  it('ignores a non-numeric snapshot index', () => {
    expect(parseSelectionFromSearch('?snapshot=abc')).toEqual({});
  });

  it('ignores a negative snapshot index', () => {
    expect(parseSelectionFromSearch('?snapshot=-1')).toEqual({});
  });

  it('ignores an empty repo param', () => {
    expect(parseSelectionFromSearch('?repo=')).toEqual({});
  });
});

describe('buildSearchFromState', () => {
  it('produces an empty string for the default live-at-latest state', () => {
    expect(buildSearchFromState({
      selectedRepoID: 'primary',
      selectedCollectionID: null,
      selectedCollectionSnapshotIndex: 0,
      liveSnapshotIndex: null,
    })).toBe('');
  });

  it('encodes a live scrub position', () => {
    expect(buildSearchFromState({
      selectedRepoID: 'primary',
      selectedCollectionID: null,
      selectedCollectionSnapshotIndex: 0,
      liveSnapshotIndex: 1,
    })).toBe('?snapshot=1');
  });

  it('encodes an archived session and its snapshot index', () => {
    expect(buildSearchFromState({
      selectedRepoID: 'primary',
      selectedCollectionID: 3,
      selectedCollectionSnapshotIndex: 2,
      liveSnapshotIndex: null,
    })).toBe('?session=3&snapshot=2');
  });

  it('encodes a non-default repo', () => {
    expect(buildSearchFromState({
      selectedRepoID: 'wt-abc',
      selectedCollectionID: null,
      selectedCollectionSnapshotIndex: 0,
      liveSnapshotIndex: null,
    })).toBe('?repo=wt-abc');
  });

  it('combines repo and session selection', () => {
    expect(buildSearchFromState({
      selectedRepoID: 'wt-abc',
      selectedCollectionID: 3,
      selectedCollectionSnapshotIndex: 0,
      liveSnapshotIndex: null,
    })).toBe('?repo=wt-abc&session=3&snapshot=0');
  });

  it('round-trips through parseSelectionFromSearch', () => {
    const state = {
      selectedRepoID: 'wt-xyz',
      selectedCollectionID: 7,
      selectedCollectionSnapshotIndex: 4,
      liveSnapshotIndex: null,
    };
    const search = buildSearchFromState(state);
    expect(parseSelectionFromSearch(search)).toEqual({
      repo: 'wt-xyz',
      collectionID: 7,
      snapshotIndex: 4,
    });
  });
});
