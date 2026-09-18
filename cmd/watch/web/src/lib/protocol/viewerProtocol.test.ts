import { describe, it, expect } from 'vitest';
import {
  normalizeGraphStreamPayload,
  normalizePersistedSessionList,
  normalizePersistedSessionDetail,
  type Snapshot,
  type Collection,
  type CommitSummary,
} from './viewerProtocol';

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

  it('carries a collection sessionId through when present, omits it when absent/invalid', () => {
    const normalized = normalizeGraphStreamPayload({
      workingSnapshots: [],
      pastCollections: [
        { id: 5, worktreeId: "main", timestamp: TIMESTAMP, snapshots: [], sessionId: 42 },
        { id: 6, worktreeId: "main", timestamp: TIMESTAMP, snapshots: [] }, // no sessionId (persistence disabled)
        { id: 7, worktreeId: "main", timestamp: TIMESTAMP, snapshots: [], sessionId: 0 }, // omitempty on the wire
      ],
    });

    expect(normalized.pastCollections[0].sessionId).toBe(42);
    expect(normalized.pastCollections[1].sessionId).toBeUndefined();
    expect(normalized.pastCollections[2].sessionId).toBeUndefined();
  });
});

describe('normalizePersistedSessionList', () => {
  it('normalizes a well-formed listing', () => {
    const list = normalizePersistedSessionList([
      {
        id: 1,
        worktreeId: "main",
        runId: 10,
        number: 1,
        createdAt: TIMESTAMP,
        closedAt: "2026-02-12T11:00:00Z",
        closedReason: "committed",
        snapshotCount: 3,
        commitCount: 1,
      },
    ]);

    expect(list).toEqual([{
      id: 1,
      worktreeId: "main",
      runId: 10,
      number: 1,
      createdAt: TIMESTAMP,
      closedAt: "2026-02-12T11:00:00Z",
      closedReason: "committed",
      snapshotCount: 3,
      commitCount: 1,
    }]);
  });

  it('drops entries missing an id or worktreeId, keeps the rest', () => {
    const list = normalizePersistedSessionList([
      { id: 1, worktreeId: "main", runId: 1, number: 1, createdAt: TIMESTAMP, snapshotCount: 0, commitCount: 0 },
      { id: 2, runId: 1, number: 2, createdAt: TIMESTAMP }, // missing worktreeId
      { worktreeId: "main", runId: 1, number: 3, createdAt: TIMESTAMP }, // missing id
      null,
      "garbage",
    ]);

    expect(list).toHaveLength(1);
    expect(list[0]!.id).toBe(1);
  });

  it('defaults closedAt to null and closedReason to empty string for an open session', () => {
    const list = normalizePersistedSessionList([
      { id: 1, worktreeId: "main", runId: 1, number: 1, createdAt: TIMESTAMP, snapshotCount: 1, commitCount: 0 },
    ]);

    expect(list[0]!.closedAt).toBeNull();
    expect(list[0]!.closedReason).toBe("");
  });

  it('returns an empty list for non-array input', () => {
    expect(normalizePersistedSessionList(null)).toEqual([]);
    expect(normalizePersistedSessionList({})).toEqual([]);
  });
});

describe('normalizePersistedSessionDetail', () => {
  it('normalizes a well-formed detail response', () => {
    const detail = normalizePersistedSessionDetail({
      id: 1,
      worktreeId: "main",
      runId: 10,
      number: 1,
      createdAt: TIMESTAMP,
      closedAt: "2026-02-12T11:00:00Z",
      closedReason: "committed",
      snapshotCount: 2,
      commitCount: 1,
      snapshots: [
        { position: 0, source: "digraph{a}", format: "dot", kind: "baseline", createdAt: TIMESTAMP },
        { position: 1, source: "digraph{a;b}", format: "dot", kind: "incremental", createdAt: TIMESTAMP },
      ],
      commits: [{ position: 0, hash: "aaa111", subject: "first" }],
    });

    expect(detail).not.toBeNull();
    expect(detail!.snapshots).toHaveLength(2);
    expect(detail!.snapshots[0]).toEqual({
      position: 0, source: "digraph{a}", format: "dot", kind: "baseline", createdAt: TIMESTAMP,
    });
    expect(detail!.commits).toEqual([{ position: 0, hash: "aaa111", subject: "first" }]);
  });

  it('drops malformed snapshot/commit entries, keeps well-formed ones', () => {
    const detail = normalizePersistedSessionDetail({
      id: 1,
      worktreeId: "main",
      runId: 10,
      number: 1,
      createdAt: TIMESTAMP,
      snapshotCount: 1,
      commitCount: 1,
      snapshots: [
        { position: 0, source: "digraph{a}", format: "dot", kind: "baseline", createdAt: TIMESTAMP },
        { position: 1 }, // missing source
      ],
      commits: [
        { position: 0, hash: "aaa111", subject: "first" },
        { position: 1, subject: "no hash" },
      ],
    });

    expect(detail!.snapshots).toHaveLength(1);
    expect(detail!.commits).toHaveLength(1);
  });

  it('returns null for a summary with no id or worktreeId', () => {
    expect(normalizePersistedSessionDetail({ snapshots: [], commits: [] })).toBeNull();
    expect(normalizePersistedSessionDetail(null)).toBeNull();
  });
});
