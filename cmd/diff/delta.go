package diff

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/LegacyCodeHQ/clarity/depgraph"
)

// SemanticAnalyzer computes optional semantic findings from two snapshots and their structural delta.
type SemanticAnalyzer func(base, target depgraph.DependencyGraph, delta graphDelta) ([]string, error)

func buildGraphDelta(base, target depgraph.DependencyGraph) (graphDelta, error) {
	baseAdj, err := depgraph.AdjacencyList(base)
	if err != nil {
		return graphDelta{}, fmt.Errorf("failed to read base adjacency: %w", err)
	}
	targetAdj, err := depgraph.AdjacencyList(target)
	if err != nil {
		return graphDelta{}, fmt.Errorf("failed to read target adjacency: %w", err)
	}

	baseNodes := collectNodes(baseAdj)
	targetNodes := collectNodes(targetAdj)

	delta := graphDelta{
		nodesAdded:   setDifference(targetNodes, baseNodes),
		nodesRemoved: setDifference(baseNodes, targetNodes),
		edgesAdded:   edgeDifference(collectEdges(targetAdj), collectEdges(baseAdj)),
		edgesRemoved: edgeDifference(collectEdges(baseAdj), collectEdges(targetAdj)),
	}

	sort.Strings(delta.nodesAdded)
	sort.Strings(delta.nodesRemoved)
	sort.Slice(delta.edgesAdded, func(i, j int) bool {
		leftFrom := filepath.Clean(delta.edgesAdded[i].from)
		rightFrom := filepath.Clean(delta.edgesAdded[j].from)
		if leftFrom == rightFrom {
			return filepath.Clean(delta.edgesAdded[i].to) < filepath.Clean(delta.edgesAdded[j].to)
		}
		return leftFrom < rightFrom
	})
	sort.Slice(delta.edgesRemoved, func(i, j int) bool {
		leftFrom := filepath.Clean(delta.edgesRemoved[i].from)
		rightFrom := filepath.Clean(delta.edgesRemoved[j].from)
		if leftFrom == rightFrom {
			return filepath.Clean(delta.edgesRemoved[i].to) < filepath.Clean(delta.edgesRemoved[j].to)
		}
		return leftFrom < rightFrom
	})

	return delta, nil
}

func applySemanticAnalyzers(base, target depgraph.DependencyGraph, delta graphDelta, analyzers []SemanticAnalyzer) (graphDelta, error) {
	if len(analyzers) == 0 {
		return delta, nil
	}

	findings := []string{}
	for _, analyzer := range analyzers {
		if analyzer == nil {
			continue
		}
		semanticFindings, err := analyzer(base, target, delta)
		if err != nil {
			return graphDelta{}, err
		}
		findings = append(findings, semanticFindings...)
	}
	sort.Strings(findings)
	delta.findings = findings
	return delta, nil
}

func collectNodes(adj map[string][]string) map[string]struct{} {
	nodes := make(map[string]struct{}, len(adj))
	for from, deps := range adj {
		nodes[from] = struct{}{}
		for _, to := range deps {
			nodes[to] = struct{}{}
		}
	}
	return nodes
}

func setDifference(left, right map[string]struct{}) []string {
	result := []string{}
	for v := range left {
		if _, ok := right[v]; !ok {
			result = append(result, v)
		}
	}
	return result
}

func collectEdges(adj map[string][]string) map[graphEdge]struct{} {
	edges := make(map[graphEdge]struct{})
	for from, deps := range adj {
		for _, to := range deps {
			edges[graphEdge{from: from, to: to}] = struct{}{}
		}
	}
	return edges
}

func edgeDifference(left, right map[graphEdge]struct{}) []graphEdge {
	result := []graphEdge{}
	for edge := range left {
		if _, ok := right[edge]; !ok {
			result = append(result, edge)
		}
	}
	return result
}

func renderSummary(delta graphDelta) string {
	var lines []string
	lines = append(lines, fmt.Sprintf("Nodes added: %d", len(delta.nodesAdded)))
	lines = append(lines, delta.nodesAdded...)
	lines = append(lines, fmt.Sprintf("Nodes removed: %d", len(delta.nodesRemoved)))
	lines = append(lines, delta.nodesRemoved...)
	lines = append(lines, fmt.Sprintf("Edges added: %d", len(delta.edgesAdded)))
	for _, e := range delta.edgesAdded {
		lines = append(lines, fmt.Sprintf("%s -> %s", e.from, e.to))
	}
	lines = append(lines, fmt.Sprintf("Edges removed: %d", len(delta.edgesRemoved)))
	for _, e := range delta.edgesRemoved {
		lines = append(lines, fmt.Sprintf("%s -> %s", e.from, e.to))
	}
	lines = append(lines, fmt.Sprintf("Semantic findings: %d", len(delta.findings)))
	lines = append(lines, delta.findings...)
	return strings.Join(lines, "\n")
}

func renderDelta(format string, delta graphDelta) (string, error) {
	formatter, err := NewDiffFormatter(format)
	if err != nil {
		return "", err
	}
	return formatter.Format(delta)
}
