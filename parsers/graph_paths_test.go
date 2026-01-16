package parsers

import (
	"sort"
	"testing"
)

func TestFindPathNodes_Linear(t *testing.T) {
	// A → B → C
	// paths(A, C) should return {A, B, C}
	graph := DependencyGraph{
		"A": {"B"},
		"B": {"C"},
		"C": {},
	}

	result := FindPathNodes(graph, []string{"A", "C"})

	expected := []string{"A", "B", "C"}
	assertGraphContainsNodes(t, result, expected)
}

func TestFindPathNodes_Diamond(t *testing.T) {
	// A → B, A → C, B → D, C → D
	// paths(A, D) should return {A, B, C, D} (both paths are shortest)
	graph := DependencyGraph{
		"A": {"B", "C"},
		"B": {"D"},
		"C": {"D"},
		"D": {},
	}

	result := FindPathNodes(graph, []string{"A", "D"})

	expected := []string{"A", "B", "C", "D"}
	assertGraphContainsNodes(t, result, expected)
}

func TestFindPathNodes_Disconnected(t *testing.T) {
	// A → B, C → D (no connection between groups)
	// paths(A, C) should return {A, C} (no intermediate nodes)
	graph := DependencyGraph{
		"A": {"B"},
		"B": {},
		"C": {"D"},
		"D": {},
	}

	result := FindPathNodes(graph, []string{"A", "C"})

	expected := []string{"A", "C"}
	assertGraphContainsNodes(t, result, expected)
}

func TestFindPathNodes_MultiFile(t *testing.T) {
	// A → B → C → D
	// paths(A, C, D) should return {A, B, C, D}
	graph := DependencyGraph{
		"A": {"B"},
		"B": {"C"},
		"C": {"D"},
		"D": {},
	}

	result := FindPathNodes(graph, []string{"A", "C", "D"})

	expected := []string{"A", "B", "C", "D"}
	assertGraphContainsNodes(t, result, expected)
}

func TestFindPathNodes_AllPaths(t *testing.T) {
	// A → B → C (short path)
	// A → D → E → C (long path)
	// paths(A, C) should return {A, B, C, D, E} (all paths, not just shortest)
	graph := DependencyGraph{
		"A": {"B", "D"},
		"B": {"C"},
		"C": {},
		"D": {"E"},
		"E": {"C"},
	}

	result := FindPathNodes(graph, []string{"A", "C"})

	// All nodes should be included since they're all on some path
	expected := []string{"A", "B", "C", "D", "E"}
	assertGraphContainsNodes(t, result, expected)
}

func TestFindPathNodes_Bidirectional(t *testing.T) {
	// A → B (directed edge A to B)
	// paths(B, A) should still find connection (treated bidirectionally)
	graph := DependencyGraph{
		"A": {"B"},
		"B": {},
	}

	result := FindPathNodes(graph, []string{"B", "A"})

	expected := []string{"A", "B"}
	assertGraphContainsNodes(t, result, expected)
}

func TestFindPathNodes_SingleTarget(t *testing.T) {
	// With only one target, just return that node
	graph := DependencyGraph{
		"A": {"B"},
		"B": {},
	}

	result := FindPathNodes(graph, []string{"A"})

	expected := []string{"A"}
	assertGraphContainsNodes(t, result, expected)

	if len(result) != 1 {
		t.Errorf("Expected exactly 1 node, got %d", len(result))
	}
}

func TestFindPathNodes_NoTargets(t *testing.T) {
	graph := DependencyGraph{
		"A": {"B"},
		"B": {},
	}

	result := FindPathNodes(graph, []string{})

	if len(result) != 0 {
		t.Errorf("Expected empty result for no targets, got %d nodes", len(result))
	}
}

func TestFindPathNodes_InvalidTarget(t *testing.T) {
	// Target "X" doesn't exist in graph
	graph := DependencyGraph{
		"A": {"B"},
		"B": {},
	}

	result := FindPathNodes(graph, []string{"A", "X"})

	// Should only include A (X doesn't exist)
	expected := []string{"A"}
	assertGraphContainsNodes(t, result, expected)
}

func TestFindPathNodes_PreservesEdges(t *testing.T) {
	// A → B → C
	// When filtering to {A, B, C}, edges should be preserved
	graph := DependencyGraph{
		"A": {"B"},
		"B": {"C"},
		"C": {},
	}

	result := FindPathNodes(graph, []string{"A", "C"})

	// Check that edges are preserved
	if deps, ok := result["A"]; ok {
		if len(deps) != 1 || deps[0] != "B" {
			t.Errorf("Expected A → B edge, got %v", deps)
		}
	} else {
		t.Error("Node A not in result")
	}

	if deps, ok := result["B"]; ok {
		if len(deps) != 1 || deps[0] != "C" {
			t.Errorf("Expected B → C edge, got %v", deps)
		}
	} else {
		t.Error("Node B not in result")
	}
}

func TestFindPathNodes_ComplexGraph(t *testing.T) {
	// More complex graph:
	//     B
	//    / \
	//   A   D → E
	//    \ /
	//     C
	// paths(A, E) should find A, B, D, E and A, C, D, E
	graph := DependencyGraph{
		"A": {"B", "C"},
		"B": {"D"},
		"C": {"D"},
		"D": {"E"},
		"E": {},
	}

	result := FindPathNodes(graph, []string{"A", "E"})

	expected := []string{"A", "B", "C", "D", "E"}
	assertGraphContainsNodes(t, result, expected)
}

func TestExtractSubgraph(t *testing.T) {
	original := DependencyGraph{
		"A": {"B", "C"},
		"B": {"C"},
		"C": {},
	}

	nodesToKeep := map[string]bool{
		"A": true,
		"B": true,
	}

	result := extractSubgraph(original, nodesToKeep)

	// Should have A and B
	if _, ok := result["A"]; !ok {
		t.Error("A should be in result")
	}
	if _, ok := result["B"]; !ok {
		t.Error("B should be in result")
	}

	// C should not be in result
	if _, ok := result["C"]; ok {
		t.Error("C should not be in result")
	}

	// A's deps should only include B (not C)
	if deps := result["A"]; len(deps) != 1 || deps[0] != "B" {
		t.Errorf("A should only have B as dep, got %v", deps)
	}

	// B's deps should be empty (C was filtered out)
	if deps := result["B"]; len(deps) != 0 {
		t.Errorf("B should have no deps, got %v", deps)
	}
}

// Helper functions

func assertGraphContainsNodes(t *testing.T, graph DependencyGraph, expectedNodes []string) {
	t.Helper()

	for _, node := range expectedNodes {
		if _, ok := graph[node]; !ok {
			t.Errorf("Expected node %s not found in graph", node)
		}
	}

	// Also check we don't have extra nodes
	var actualNodes []string
	for node := range graph {
		actualNodes = append(actualNodes, node)
	}

	sort.Strings(actualNodes)
	sort.Strings(expectedNodes)

	if len(actualNodes) != len(expectedNodes) {
		t.Errorf("Expected %d nodes %v, got %d nodes %v", len(expectedNodes), expectedNodes, len(actualNodes), actualNodes)
	}
}

