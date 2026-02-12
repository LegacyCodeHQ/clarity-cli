import { Graphviz } from "https://cdn.jsdelivr.net/npm/@hpcc-js/wasm-graphviz@1.6.1/dist/index.js";

const graphviz = await Graphviz.load();
const container = document.getElementById("graph-container");
const statusEl = document.getElementById("status");
const statusText = document.getElementById("status-text");
const sliderEl = document.getElementById("timeline-slider");
const liveBtnEl = document.getElementById("timeline-live");
const modeEl = document.getElementById("timeline-mode");
const metaEl = document.getElementById("timeline-meta");
const sourceEl = document.getElementById("snapshot-source");

let workingSnapshots = [];
let pastCollections = [];
let selectedCollectionID = null;
let selectedCollectionSnapshotIndex = 0;
let liveSnapshotIndex = null;

function renderGraph(dot) {
  try {
    const svg = graphviz.layout(dot, "svg", "dot");
    container.innerHTML = svg;
    statusText.textContent = "Connected";
    statusEl.classList.remove("disconnected");
  } catch (err) {
    console.error("Graphviz render error:", err);
    statusText.textContent = "Render error";
  }
}

function renderWaitingState() {
  container.innerHTML = '<p id="placeholder">No uncommitted changes. Waiting for file changes...</p>';
}

function formatSnapshotMeta(snapshot, index, total) {
  const time = new Date(snapshot.timestamp);
  return `#${index + 1}/${total} | id ${snapshot.id} | ${time.toLocaleTimeString()}`;
}

function getSelectedCollection() {
  if (selectedCollectionID === null) {
    return null;
  }
  return pastCollections.find((collection) => collection.id === selectedCollectionID) || null;
}

function syncSourceSelector() {
  const previousValue = sourceEl.value;
  sourceEl.innerHTML = "";

  const liveOption = document.createElement("option");
  liveOption.value = "live";
  liveOption.textContent = "Current working directory (live)";
  sourceEl.appendChild(liveOption);

  const orderedCollections = [...pastCollections].reverse();
  orderedCollections.forEach((collection, index) => {
    const option = document.createElement("option");
    option.value = `collection:${collection.id}`;
    const time = new Date(collection.timestamp).toLocaleTimeString();
    const number = pastCollections.length - index;
    option.textContent = `Collection ${number} (${collection.snapshots.length} snapshots, ${time})`;
    sourceEl.appendChild(option);
  });

  if (selectedCollectionID === null) {
    sourceEl.value = "live";
    return;
  }

  const wantedValue = `collection:${selectedCollectionID}`;
  const wantedOption = sourceEl.querySelector(`option[value="${wantedValue}"]`);
  if (wantedOption) {
    sourceEl.value = wantedValue;
    return;
  }

  sourceEl.value = previousValue === "" ? "live" : "live";
}

function syncLiveUI() {
  const total = workingSnapshots.length;
  const latestIndex = total > 0 ? total - 1 : 0;
  if (liveSnapshotIndex !== null) {
    liveSnapshotIndex = Math.max(0, Math.min(liveSnapshotIndex, latestIndex));
    if (liveSnapshotIndex === latestIndex) {
      liveSnapshotIndex = null;
    }
  }

  const selectedIndex = liveSnapshotIndex === null ? latestIndex : liveSnapshotIndex;
  modeEl.textContent = liveSnapshotIndex === null ? "Working directory (live)" : "Working directory snapshot";
  sliderEl.disabled = total <= 1;
  sliderEl.max = total > 0 ? String(total - 1) : "0";
  sliderEl.value = total > 0 ? String(selectedIndex) : "0";
  liveBtnEl.disabled = total === 0 || liveSnapshotIndex === null;

  if (total === 0) {
    metaEl.textContent = "0 working snapshots";
    return;
  }

  metaEl.textContent = `${total} working snapshots | ${formatSnapshotMeta(workingSnapshots[selectedIndex], selectedIndex, total)}`;
}

function syncCollectionUI(collection) {
  const snapshots = collection.snapshots || [];
  const total = snapshots.length;
  if (total === 0) {
    sliderEl.disabled = true;
    sliderEl.max = "0";
    sliderEl.value = "0";
    modeEl.textContent = "Snapshot collection";
    liveBtnEl.disabled = false;
    metaEl.textContent = "Collection is empty";
    return;
  }

  selectedCollectionSnapshotIndex = Math.max(0, Math.min(selectedCollectionSnapshotIndex, total - 1));
  sliderEl.disabled = total <= 1;
  sliderEl.max = String(total - 1);
  sliderEl.value = String(selectedCollectionSnapshotIndex);
  modeEl.textContent = "Snapshot collection";
  liveBtnEl.disabled = false;
  metaEl.textContent = `${total} snapshots | ${formatSnapshotMeta(snapshots[selectedCollectionSnapshotIndex], selectedCollectionSnapshotIndex, total)}`;
}

function renderSelection() {
  syncSourceSelector();

  if (selectedCollectionID === null) {
    syncLiveUI();
    if (workingSnapshots.length === 0) {
      renderWaitingState();
      return;
    }
    const latestIndex = workingSnapshots.length - 1;
    const selectedIndex = liveSnapshotIndex === null
      ? latestIndex
      : Math.max(0, Math.min(liveSnapshotIndex, latestIndex));
    renderGraph(workingSnapshots[selectedIndex].dot);
    return;
  }

  const selectedCollection = getSelectedCollection();
  if (!selectedCollection) {
    selectedCollectionID = null;
    selectedCollectionSnapshotIndex = 0;
    renderSelection();
    return;
  }

  syncCollectionUI(selectedCollection);
  if ((selectedCollection.snapshots || []).length === 0) {
    renderWaitingState();
    return;
  }

  renderGraph(selectedCollection.snapshots[selectedCollectionSnapshotIndex].dot);
}

function mergePayload(payload) {
  workingSnapshots = payload.workingSnapshots || [];
  pastCollections = payload.pastCollections || [];

  if (workingSnapshots.length === 0) {
    liveSnapshotIndex = null;
  }

  if (selectedCollectionID !== null && !getSelectedCollection()) {
    selectedCollectionID = null;
    selectedCollectionSnapshotIndex = 0;
  }

  renderSelection();
}

sliderEl.addEventListener("input", function() {
  if (selectedCollectionID === null) {
    if (workingSnapshots.length === 0) {
      return;
    }
    const idx = Math.max(0, Math.min(Number(sliderEl.value || "0"), workingSnapshots.length - 1));
    const latestIndex = workingSnapshots.length - 1;
    liveSnapshotIndex = idx === latestIndex ? null : idx;
    renderSelection();
    return;
  }

  const collection = getSelectedCollection();
  if (!collection || (collection.snapshots || []).length === 0) {
    return;
  }

  selectedCollectionSnapshotIndex = Math.max(
    0,
    Math.min(Number(sliderEl.value || "0"), collection.snapshots.length - 1),
  );
  renderSelection();
});

liveBtnEl.addEventListener("click", function() {
  liveSnapshotIndex = null;
  selectedCollectionID = null;
  selectedCollectionSnapshotIndex = 0;
  renderSelection();
});

sourceEl.addEventListener("change", function(event) {
  const selected = event.target.value;
  if (selected === "live") {
    liveSnapshotIndex = null;
    selectedCollectionID = null;
    selectedCollectionSnapshotIndex = 0;
    renderSelection();
    return;
  }

  if (!selected.startsWith("collection:")) {
    liveSnapshotIndex = null;
    selectedCollectionID = null;
    selectedCollectionSnapshotIndex = 0;
    renderSelection();
    return;
  }

  const selectedID = Number(selected.split(":")[1]);
  if (!Number.isFinite(selectedID)) {
    liveSnapshotIndex = null;
    selectedCollectionID = null;
    selectedCollectionSnapshotIndex = 0;
    renderSelection();
    return;
  }

  selectedCollectionID = selectedID;
  selectedCollectionSnapshotIndex = 0;
  renderSelection();
});

function connectSSE() {
  const source = new EventSource("/events");

  source.addEventListener("graph", function(event) {
    try {
      const payload = JSON.parse(event.data);
      mergePayload(payload);
    } catch (err) {
      console.error("Invalid graph payload:", err);
      statusText.textContent = "Payload error";
    }
  });

  source.addEventListener("open", function() {
    statusText.textContent = "Connected";
    statusEl.classList.remove("disconnected");
  });

  source.addEventListener("error", function() {
    statusText.textContent = "Reconnecting...";
    statusEl.classList.add("disconnected");
  });
}

connectSSE();
