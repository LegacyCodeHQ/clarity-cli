import assert from "node:assert/strict";
import test from "node:test";

import {
  applyLiveSelection,
  applySliderInput,
  applySourceSelection,
  formatSnapshotMeta,
  getViewModel,
  mergePayload,
} from "./viewer_state.mjs";

const TIMESTAMP = "2026-02-12T10:00:00Z";

function snapshot(id, dot = `digraph ${id} {}`) {
  return { id, timestamp: TIMESTAMP, dot };
}

function collection(id, snapshots) {
  return {
    id,
    timestamp: TIMESTAMP,
    snapshots,
  };
}

function baseState() {
  return {
    workingSnapshots: [],
    pastCollections: [],
    selectedCollectionID: null,
    selectedCollectionSnapshotIndex: 0,
    liveSnapshotIndex: null,
  };
}

test("mergePayload resets live snapshot index when no working snapshots remain", () => {
  const state = {
    ...baseState(),
    workingSnapshots: [snapshot(1), snapshot(2)],
    liveSnapshotIndex: 0,
  };

  const next = mergePayload(state, {
    workingSnapshots: [],
    pastCollections: [],
  });

  assert.equal(next.liveSnapshotIndex, null);
  assert.deepEqual(next.workingSnapshots, []);
});

test("mergePayload falls back to live mode when selected collection disappears", () => {
  const state = {
    ...baseState(),
    selectedCollectionID: 42,
    selectedCollectionSnapshotIndex: 3,
  };

  const next = mergePayload(state, {
    workingSnapshots: [snapshot(7)],
    pastCollections: [collection(1, [snapshot(10)])],
  });

  assert.equal(next.selectedCollectionID, null);
  assert.equal(next.selectedCollectionSnapshotIndex, 0);
});

test("applySourceSelection returns to live mode for invalid source values", () => {
  const state = {
    ...baseState(),
    selectedCollectionID: 5,
    selectedCollectionSnapshotIndex: 2,
    liveSnapshotIndex: 1,
  };

  const invalid = applySourceSelection(state, "bad-value");
  assert.equal(invalid.selectedCollectionID, null);
  assert.equal(invalid.selectedCollectionSnapshotIndex, 0);
  assert.equal(invalid.liveSnapshotIndex, null);

  const nanID = applySourceSelection(state, "collection:abc");
  assert.equal(nanID.selectedCollectionID, null);
  assert.equal(nanID.selectedCollectionSnapshotIndex, 0);
});

test("applySourceSelection selects finite collection id and resets collection index", () => {
  const state = {
    ...baseState(),
    selectedCollectionSnapshotIndex: 4,
    pastCollections: [collection(123, [snapshot(1), snapshot(2)])],
  };

  const next = applySourceSelection(state, "collection:123");

  assert.equal(next.selectedCollectionID, 123);
  assert.equal(next.selectedCollectionSnapshotIndex, 0);
});

test("applySliderInput in live mode clamps and maps latest index to null", () => {
  const state = {
    ...baseState(),
    workingSnapshots: [snapshot(1), snapshot(2), snapshot(3)],
  };

  const older = applySliderInput(state, "1");
  assert.equal(older.liveSnapshotIndex, 1);

  const latest = applySliderInput(state, "50");
  assert.equal(latest.liveSnapshotIndex, null);

  const negative = applySliderInput(state, "-8");
  assert.equal(negative.liveSnapshotIndex, 0);
});

test("applySliderInput in collection mode clamps snapshot index", () => {
  const state = {
    ...baseState(),
    selectedCollectionID: 9,
    selectedCollectionSnapshotIndex: 1,
    pastCollections: [collection(9, [snapshot(1), snapshot(2), snapshot(3)])],
  };

  const high = applySliderInput(state, "20");
  assert.equal(high.selectedCollectionSnapshotIndex, 2);

  const low = applySliderInput(state, "-1");
  assert.equal(low.selectedCollectionSnapshotIndex, 0);
});

test("getViewModel returns waiting state metadata for empty live snapshots", () => {
  const vm = getViewModel(baseState(), () => "10:00:00");
  assert.equal(vm.renderDot, null);
  assert.equal(vm.timeline.modeText, "Working directory (live)");
  assert.equal(vm.timeline.metaText, "0 working snapshots");
  assert.equal(vm.timeline.sliderDisabled, true);
  assert.equal(vm.sourceValue, "live");
});

test("getViewModel returns selected live snapshot dot and metadata", () => {
  const state = {
    ...baseState(),
    workingSnapshots: [snapshot(1, "digraph one {}"), snapshot(2, "digraph two {}")],
    liveSnapshotIndex: 0,
  };

  const vm = getViewModel(state, () => "10:00:00");
  assert.equal(vm.renderDot, "digraph one {}");
  assert.equal(vm.timeline.modeText, "Working directory snapshot");
  assert.equal(
    vm.timeline.metaText,
    "2 working snapshots | #1/2 | id 1 | 10:00:00",
  );
});

test("applyLiveSelection resets to live mode baseline", () => {
  const state = {
    ...baseState(),
    selectedCollectionID: 8,
    selectedCollectionSnapshotIndex: 2,
    liveSnapshotIndex: 1,
  };

  const next = applyLiveSelection(state);

  assert.equal(next.selectedCollectionID, null);
  assert.equal(next.selectedCollectionSnapshotIndex, 0);
  assert.equal(next.liveSnapshotIndex, null);
});

test("formatSnapshotMeta renders snapshot position, id, and time", () => {
  const result = formatSnapshotMeta(snapshot(99), 1, 3, () => "11:11:11");
  assert.equal(result, "#2/3 | id 99 | 11:11:11");
});

