function clamp(value, min, max) {
  return Math.max(min, Math.min(value, max));
}

function formatTime(timestamp) {
  return new Date(timestamp).toLocaleTimeString();
}

export function formatSnapshotMeta(snapshot, index, total, timeFormatter = formatTime) {
  return `#${index + 1}/${total} | id ${snapshot.id} | ${timeFormatter(snapshot.timestamp)}`;
}

export function getSelectedCollection(state) {
  if (state.selectedCollectionID === null) {
    return null;
  }
  return state.pastCollections.find((collection) => collection.id === state.selectedCollectionID) || null;
}

export function normalizeState(state) {
  const next = {
    workingSnapshots: Array.isArray(state.workingSnapshots) ? state.workingSnapshots : [],
    pastCollections: Array.isArray(state.pastCollections) ? state.pastCollections : [],
    selectedCollectionID: state.selectedCollectionID ?? null,
    selectedCollectionSnapshotIndex: Number.isFinite(state.selectedCollectionSnapshotIndex)
      ? state.selectedCollectionSnapshotIndex
      : 0,
    liveSnapshotIndex: state.liveSnapshotIndex === null || Number.isFinite(state.liveSnapshotIndex)
      ? state.liveSnapshotIndex
      : null,
  };

  if (next.workingSnapshots.length === 0) {
    next.liveSnapshotIndex = null;
  }

  const selectedCollection = getSelectedCollection(next);
  if (next.selectedCollectionID !== null && !selectedCollection) {
    next.selectedCollectionID = null;
    next.selectedCollectionSnapshotIndex = 0;
  }

  if (next.selectedCollectionID === null) {
    const total = next.workingSnapshots.length;
    const latestIndex = total > 0 ? total - 1 : 0;
    if (next.liveSnapshotIndex !== null) {
      next.liveSnapshotIndex = clamp(next.liveSnapshotIndex, 0, latestIndex);
      if (next.liveSnapshotIndex === latestIndex) {
        next.liveSnapshotIndex = null;
      }
    }
    return next;
  }

  const collection = getSelectedCollection(next);
  const snapshots = collection ? collection.snapshots || [] : [];
  if (snapshots.length === 0) {
    next.selectedCollectionSnapshotIndex = 0;
    return next;
  }
  next.selectedCollectionSnapshotIndex = clamp(
    next.selectedCollectionSnapshotIndex,
    0,
    snapshots.length - 1,
  );
  return next;
}

export function mergePayload(state, payload) {
  return normalizeState({
    ...state,
    workingSnapshots: payload.workingSnapshots || [],
    pastCollections: payload.pastCollections || [],
  });
}

export function applySliderInput(state, rawValue) {
  const next = normalizeState(state);
  if (next.selectedCollectionID === null) {
    if (next.workingSnapshots.length === 0) {
      return next;
    }
    const latestIndex = next.workingSnapshots.length - 1;
    const idx = clamp(Number(rawValue || "0"), 0, latestIndex);
    next.liveSnapshotIndex = idx === latestIndex ? null : idx;
    return normalizeState(next);
  }

  const collection = getSelectedCollection(next);
  const snapshots = collection ? collection.snapshots || [] : [];
  if (snapshots.length === 0) {
    return next;
  }
  next.selectedCollectionSnapshotIndex = clamp(Number(rawValue || "0"), 0, snapshots.length - 1);
  return normalizeState(next);
}

export function applyLiveSelection(state) {
  return normalizeState({
    ...state,
    liveSnapshotIndex: null,
    selectedCollectionID: null,
    selectedCollectionSnapshotIndex: 0,
  });
}

export function applySourceSelection(state, selected) {
  if (selected === "live") {
    return applyLiveSelection(state);
  }
  if (!selected.startsWith("collection:")) {
    return applyLiveSelection(state);
  }

  const selectedID = Number(selected.split(":")[1]);
  if (!Number.isFinite(selectedID)) {
    return applyLiveSelection(state);
  }

  return normalizeState({
    ...state,
    selectedCollectionID: selectedID,
    selectedCollectionSnapshotIndex: 0,
  });
}

export function getSourceOptions(state, timeFormatter = formatTime) {
  const liveOption = {
    value: "live",
    text: "Current working directory (live)",
  };
  const orderedCollections = [...state.pastCollections].reverse();
  const collectionOptions = orderedCollections.map((collection, index) => {
    const number = state.pastCollections.length - index;
    const snapshots = collection.snapshots || [];
    return {
      value: `collection:${collection.id}`,
      text: `Collection ${number} (${snapshots.length} snapshots, ${timeFormatter(collection.timestamp)})`,
    };
  });

  return [liveOption, ...collectionOptions];
}

export function getViewModel(state, timeFormatter = formatTime) {
  const normalized = normalizeState(state);
  const sourceValue = normalized.selectedCollectionID === null
    ? "live"
    : `collection:${normalized.selectedCollectionID}`;

  if (normalized.selectedCollectionID === null) {
    const total = normalized.workingSnapshots.length;
    const latestIndex = total > 0 ? total - 1 : 0;
    const selectedIndex = normalized.liveSnapshotIndex === null
      ? latestIndex
      : normalized.liveSnapshotIndex;

    return {
      state: normalized,
      sourceValue,
      sourceOptions: getSourceOptions(normalized, timeFormatter),
      renderDot: total > 0 ? normalized.workingSnapshots[selectedIndex].dot : null,
      timeline: {
        modeText: normalized.liveSnapshotIndex === null
          ? "Working directory (live)"
          : "Working directory snapshot",
        sliderDisabled: total <= 1,
        sliderMax: total > 0 ? String(total - 1) : "0",
        sliderValue: total > 0 ? String(selectedIndex) : "0",
        liveButtonDisabled: total === 0 || normalized.liveSnapshotIndex === null,
        metaText: total === 0
          ? "0 working snapshots"
          : `${total} working snapshots | ${formatSnapshotMeta(
            normalized.workingSnapshots[selectedIndex],
            selectedIndex,
            total,
            timeFormatter,
          )}`,
      },
    };
  }

  const selectedCollection = getSelectedCollection(normalized);
  const snapshots = selectedCollection ? selectedCollection.snapshots || [] : [];
  const total = snapshots.length;

  return {
    state: normalized,
    sourceValue,
    sourceOptions: getSourceOptions(normalized, timeFormatter),
    renderDot: total > 0 ? snapshots[normalized.selectedCollectionSnapshotIndex].dot : null,
    timeline: {
      modeText: "Snapshot collection",
      sliderDisabled: total <= 1,
      sliderMax: total > 0 ? String(total - 1) : "0",
      sliderValue: total > 0 ? String(normalized.selectedCollectionSnapshotIndex) : "0",
      liveButtonDisabled: false,
      metaText: total === 0
        ? "Collection is empty"
        : `${total} snapshots | ${formatSnapshotMeta(
          snapshots[normalized.selectedCollectionSnapshotIndex],
          normalized.selectedCollectionSnapshotIndex,
          total,
          timeFormatter,
        )}`,
    },
  };
}

