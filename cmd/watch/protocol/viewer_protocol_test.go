package protocol

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProtocolConstants_AreStable(t *testing.T) {
	assert.Equal(t, "/", RouteIndex)
	assert.Equal(t, "/events", RouteEvents)
	assert.Equal(t, "graph", SSEEventGraph)
}

func TestWorktreeDescriptor_JSONContract(t *testing.T) {
	main := WorktreeDescriptor{ID: "main", Path: "/repo", Label: "repo", Kind: WorktreeKindMain, Active: true}
	linked := WorktreeDescriptor{ID: "wt-abc12345", Path: "/tmp/feat", Label: "feat", Kind: WorktreeKindLinked, Active: true}

	rawMain, err := json.Marshal(main)
	require.NoError(t, err)
	var mainDoc map[string]any
	require.NoError(t, json.Unmarshal(rawMain, &mainDoc))
	assert.Equal(t, "main", mainDoc["kind"])
	assert.NotContains(t, mainDoc, "isPrimary")

	rawLinked, err := json.Marshal(linked)
	require.NoError(t, err)
	var linkedDoc map[string]any
	require.NoError(t, json.Unmarshal(rawLinked, &linkedDoc))
	assert.Equal(t, "linked", linkedDoc["kind"])
}

func TestGraphStreamPayload_JSONContract(t *testing.T) {
	ts := time.Date(2026, 2, 12, 10, 0, 0, 0, time.UTC)
	payload := GraphStreamPayload{
		WorkingSnapshots: []GraphSnapshot{
			{ID: 1, Timestamp: ts, DOT: "digraph { A -> B; }"},
		},
		PastCollections: []SnapshotCollection{
			{
				ID:        7,
				Timestamp: ts,
				Snapshots: []GraphSnapshot{
					{ID: 2, Timestamp: ts, DOT: "digraph { X -> Y; }"},
				},
			},
		},
		LatestWorkingID:        1,
		LatestPastCollectionID: 7,
	}

	raw, err := json.Marshal(payload)
	require.NoError(t, err)

	var doc map[string]any
	require.NoError(t, json.Unmarshal(raw, &doc))

	assert.Contains(t, doc, "workingSnapshots")
	assert.Contains(t, doc, "pastCollections")
	assert.Contains(t, doc, "latestWorkingId")
	assert.Contains(t, doc, "latestPastCollectionId")

	working, ok := doc["workingSnapshots"].([]any)
	require.True(t, ok)
	require.Len(t, working, 1)

	first, ok := working[0].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, first, "id")
	assert.Contains(t, first, "timestamp")
	assert.Contains(t, first, "dot")
	assert.Equal(t, "digraph { A -> B; }", first["dot"])
}
