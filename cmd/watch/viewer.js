import { Graphviz } from "https://cdn.jsdelivr.net/npm/@hpcc-js/wasm-graphviz@1.6.1/dist/index.js";
import {
  applyLiveSelection,
  applySliderInput,
  applySourceSelection,
  getViewModel,
  mergePayload,
} from "./viewer_state.mjs";

const graphviz = await Graphviz.load();
const container = document.getElementById("graph-container");
const statusEl = document.getElementById("status");
const statusText = document.getElementById("status-text");
const sliderEl = document.getElementById("timeline-slider");
const liveBtnEl = document.getElementById("timeline-live");
const modeEl = document.getElementById("timeline-mode");
const metaEl = document.getElementById("timeline-meta");
const sourceEl = document.getElementById("snapshot-source");

let state = {
  workingSnapshots: [],
  pastCollections: [],
  selectedCollectionID: null,
  selectedCollectionSnapshotIndex: 0,
  liveSnapshotIndex: null,
};

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

function syncSourceSelector(options, selectedValue) {
  sourceEl.innerHTML = "";
  options.forEach((sourceOption) => {
    const option = document.createElement("option");
    option.value = sourceOption.value;
    option.textContent = sourceOption.text;
    sourceEl.appendChild(option);
  });
  sourceEl.value = selectedValue;
}

function renderSelection() {
  const vm = getViewModel(state);
  state = vm.state;

  syncSourceSelector(vm.sourceOptions, vm.sourceValue);
  modeEl.textContent = vm.timeline.modeText;
  sliderEl.disabled = vm.timeline.sliderDisabled;
  sliderEl.max = vm.timeline.sliderMax;
  sliderEl.value = vm.timeline.sliderValue;
  liveBtnEl.disabled = vm.timeline.liveButtonDisabled;
  metaEl.textContent = vm.timeline.metaText;

  if (!vm.renderDot) {
    renderWaitingState();
    return;
  }
  renderGraph(vm.renderDot);
}

sliderEl.addEventListener("input", function() {
  state = applySliderInput(state, sliderEl.value || "0");
  renderSelection();
});

liveBtnEl.addEventListener("click", function() {
  state = applyLiveSelection(state);
  renderSelection();
});

sourceEl.addEventListener("change", function(event) {
  state = applySourceSelection(state, event.target.value);
  renderSelection();
});

function connectSSE() {
  const source = new EventSource("/events");

  source.addEventListener("graph", function(event) {
    try {
      const payload = JSON.parse(event.data);
      state = mergePayload(state, payload);
      renderSelection();
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

renderSelection();
connectSSE();
