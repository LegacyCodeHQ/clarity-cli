import { describe, it, expect } from 'vitest';
import { parseSelectionFromSearch, buildSearchFromState } from './urlSelection';

describe('parseSelectionFromSearch', () => {
  it('returns an empty selection for no query string', () => {
    expect(parseSelectionFromSearch('')).toEqual({});
  });

  it('parses worktree, session, and snapshot together', () => {
    expect(parseSelectionFromSearch('?worktree=wt-abc&session=3&snapshot=2')).toEqual({
      worktree: 'wt-abc',
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

  it('ignores an empty worktree param', () => {
    expect(parseSelectionFromSearch('?worktree=')).toEqual({});
  });
});

describe('buildSearchFromState', () => {
  it('produces an empty string for the default live-at-latest state', () => {
    expect(buildSearchFromState({
      selectedWorktreeID: 'main',
      selectedCollectionID: null,
      selectedCollectionSnapshotIndex: 0,
      liveSnapshotIndex: null,
    })).toBe('');
  });

  it('encodes a live scrub position', () => {
    expect(buildSearchFromState({
      selectedWorktreeID: 'main',
      selectedCollectionID: null,
      selectedCollectionSnapshotIndex: 0,
      liveSnapshotIndex: 1,
    })).toBe('?snapshot=1');
  });

  it('encodes an archived session and its snapshot index', () => {
    expect(buildSearchFromState({
      selectedWorktreeID: 'main',
      selectedCollectionID: 3,
      selectedCollectionSnapshotIndex: 2,
      liveSnapshotIndex: null,
    })).toBe('?session=3&snapshot=2');
  });

  it('encodes a non-default worktree', () => {
    expect(buildSearchFromState({
      selectedWorktreeID: 'wt-abc',
      selectedCollectionID: null,
      selectedCollectionSnapshotIndex: 0,
      liveSnapshotIndex: null,
    })).toBe('?worktree=wt-abc');
  });

  it('combines worktree and session selection', () => {
    expect(buildSearchFromState({
      selectedWorktreeID: 'wt-abc',
      selectedCollectionID: 3,
      selectedCollectionSnapshotIndex: 0,
      liveSnapshotIndex: null,
    })).toBe('?worktree=wt-abc&session=3&snapshot=0');
  });

  it('round-trips through parseSelectionFromSearch', () => {
    const state = {
      selectedWorktreeID: 'wt-xyz',
      selectedCollectionID: 7,
      selectedCollectionSnapshotIndex: 4,
      liveSnapshotIndex: null,
    };
    const search = buildSearchFromState(state);
    expect(parseSelectionFromSearch(search)).toEqual({
      worktree: 'wt-xyz',
      collectionID: 7,
      snapshotIndex: 4,
    });
  });
});
