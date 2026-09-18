import { describe, it, expect } from 'vitest';
import { normalizeGraphStreamPayload, type Snapshot, type Collection, type CommitSummary } from './viewerProtocol';

const TIMESTAMP = "2026-02-12T10:00:00Z";

function snapshot(id: number, worktreeId = "main", dot = `digraph ${id} {}`): Snapshot {
  return { id, worktreeId, timestamp: TIMESTAMP, dot };
}

function collection(
  id: number,
  snapshots: Snapshot[],
  worktreeId = "main",
  commitHistory: CommitSummary[] = []
): Collection {
  return {
    id,
    worktreeId,
    timestamp: TIMESTAMP,
    snapshots,
    commitHistory,
  };
}

describe('normalizeGraphStreamPayload', () => {
  it('filters malformed snapshot and collection data', () => {
    const normalized = normalizeGraphStreamPayload({
      worktrees: [{ id: "main", path: "/repo", label: "repo", kind: 'main' }],
      workingSnapshots: [
        snapshot(1),
        { id: 2, worktreeId: "main", timestamp: TIMESTAMP }, // missing dot
        null,
      ],
      pastCollections: [
        collection(10, [snapshot(7), { id: 8, worktreeId: "main", timestamp: TIMESTAMP }]), // inner snapshot missing dot
        { id: 11, worktreeId: "main", timestamp: TIMESTAMP }, // missing snapshots array
        null,
      ],
      latestWorkingId: "bad", // not a number
      latestPastCollectionId: 22,
    });

    expect(normalized.workingSnapshots).toEqual([snapshot(1)]);
    expect(normalized.pastCollections).toEqual([collection(10, [snapshot(7)])]);
    expect(normalized.latestWorkingId).toBe(0);
    expect(normalized.latestPastCollectionId).toBe(22);
  });

  it('handles non-object input', () => {
    expect(normalizeGraphStreamPayload(null)).toEqual({
      worktrees: [],
      format: "dot",
      workingSnapshots: [],
      pastCollections: [],
    });
  });

  it('defaults format to "dot" when missing and preserves it when present', () => {
    expect(normalizeGraphStreamPayload({ workingSnapshots: [], pastCollections: [] }).format).toBe("dot");
    expect(
      normalizeGraphStreamPayload({ format: "mermaid", workingSnapshots: [], pastCollections: [] }).format,
    ).toBe("mermaid");
  });

  it('defaults worktreeId to empty string when missing', () => {
    const normalized = normalizeGraphStreamPayload({
      workingSnapshots: [{ id: 1, timestamp: TIMESTAMP, dot: "digraph {}" }],
      pastCollections: [{
        id: 5,
        timestamp: TIMESTAMP,
        snapshots: [{ id: 2, timestamp: TIMESTAMP, dot: "digraph {}" }],
      }],
    });

    expect(normalized.workingSnapshots[0].worktreeId).toBe("");
    expect(normalized.pastCollections[0].worktreeId).toBe("");
    expect(normalized.pastCollections[0].snapshots[0].worktreeId).toBe("");
  });

  it('normalizes the worktrees[] tab descriptor list', () => {
    const normalized = normalizeGraphStreamPayload({
      worktrees: [
        { id: "main", path: "/repo", label: "clarity-cli", kind: 'main' },
        { id: "wt-abc12345", path: "/tmp/feat", label: "clarity-cli (feat)", kind: 'linked' },
        { id: "" }, // empty id should be dropped
        null,
        "garbage",
      ],
      workingSnapshots: [],
      pastCollections: [],
    });

    expect(normalized.worktrees).toEqual([
      { id: "main", path: "/repo", label: "clarity-cli", kind: 'main', active: true },
      { id: "wt-abc12345", path: "/tmp/feat", label: "clarity-cli (feat)", kind: 'linked', active: true },
    ]);
  });

  it('falls back to worktree id as label when label missing', () => {
    const normalized = normalizeGraphStreamPayload({
      worktrees: [{ id: "main", path: "/repo", kind: 'main' }],
      workingSnapshots: [],
      pastCollections: [],
    });

    expect(normalized.worktrees[0].label).toBe("main");
    expect(normalized.worktrees[0].kind).toBe("main");
  });

  it('normalizes the active flag, defaulting missing to true', () => {
    const normalized = normalizeGraphStreamPayload({
      worktrees: [
        { id: "main", path: "/repo", label: "repo", kind: 'main', active: true },
        { id: "wt-finished", path: "/tmp/done", label: "done", kind: 'linked', active: false },
        { id: "wt-legacy", path: "/tmp/old", label: "old", kind: 'linked' }, // no active field
      ],
      workingSnapshots: [],
      pastCollections: [],
    });

    expect(normalized.worktrees[0].active).toBe(true);
    expect(normalized.worktrees[1].active).toBe(false);
    // Backward tolerance: a descriptor without `active` is treated as active.
    expect(normalized.worktrees[2].active).toBe(true);
  });

  it('carries the sessionStart flag through normalization', () => {
    const normalized = normalizeGraphStreamPayload({
      workingSnapshots: [
        { id: 1, worktreeId: "main", timestamp: TIMESTAMP, dot: "digraph {}", sessionStart: true },
        { id: 2, worktreeId: "main", timestamp: TIMESTAMP, dot: "digraph {}" },
      ],
      pastCollections: [],
    });

    expect(normalized.workingSnapshots[0].sessionStart).toBe(true);
    // Absent/false flag stays falsy (omitted, matching the backend's omitempty).
    expect(normalized.workingSnapshots[1].sessionStart).toBeFalsy();
  });

  it('carries commit history through archived collection normalization', () => {
    const commit = {
      hash: "1234567890abcdef",
      shortHash: "1234567",
      subject: "add timeline shortcut",
      author: "Test User",
      email: "test@example.com",
      timestamp: TIMESTAMP,
    };

    const normalized = normalizeGraphStreamPayload({
      workingSnapshots: [],
      pastCollections: [{
        id: 5,
        worktreeId: "main",
        timestamp: TIMESTAMP,
        snapshots: [snapshot(1)],
        commitHistory: [commit, { subject: "missing hash" }],
      }],
    });

    expect(normalized.pastCollections).toEqual([collection(5, [snapshot(1)], "main", [commit])]);
  });
});
