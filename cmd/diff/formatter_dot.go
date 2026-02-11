package diff

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

func (dotDiffFormatter) Format(delta graphDelta) (string, error) {
	return renderDeltaDOT(delta), nil
}

func renderDeltaDOT(delta graphDelta) string {
	var b strings.Builder
	b.WriteString("digraph diff {\n")
	b.WriteString("  rankdir=LR;\n")
	b.WriteString("  node [shape=box];\n")

	changedNodes := sortedChangedNodes(delta.changedNodes)
	for _, n := range changedNodes {
		b.WriteString(fmt.Sprintf("  %q [label=%q, style=filled, fillcolor=\"#d9f2d9\", color=\"#2e8b57\"];\n", n, filepath.Base(n)))
	}
	for _, n := range delta.nodesAdded {
		b.WriteString(fmt.Sprintf("  %q [label=%q, style=filled, fillcolor=\"#d9f2d9\", color=\"#2e8b57\"];\n", n, filepath.Base(n)))
	}
	for _, n := range delta.nodesRemoved {
		b.WriteString(fmt.Sprintf("  %q [label=%q, style=filled, fillcolor=\"#f8d7da\", color=\"#b22222\"];\n", n, filepath.Base(n)))
	}

	for _, e := range delta.edgesAdded {
		b.WriteString(fmt.Sprintf("  %q -> %q [color=\"#2e8b57\"];\n", e.from, e.to))
	}
	for _, e := range delta.edgesRemoved {
		b.WriteString(fmt.Sprintf("  %q -> %q [color=\"#b22222\", style=dashed];\n", e.from, e.to))
	}

	b.WriteString("}\n")
	return b.String()
}

func sortedChangedNodes(changed map[string]struct{}) []string {
	if len(changed) == 0 {
		return nil
	}
	nodes := make([]string, 0, len(changed))
	for n := range changed {
		nodes = append(nodes, n)
	}
	sort.Strings(nodes)
	return nodes
}
