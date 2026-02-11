package diff

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

func (mermaidDiffFormatter) Format(delta graphDelta) (string, error) {
	return renderDeltaMermaid(delta), nil
}

func renderDeltaMermaid(delta graphDelta) string {
	var b strings.Builder
	b.WriteString("flowchart LR\n")

	nodeIDs := make(map[string]string)
	nodes := sortedChangedNodes(delta.changedNodes)
	nodes = append(nodes, delta.nodesAdded...)
	nodes = append(nodes, delta.nodesRemoved...)
	nodes = dedupeSortedStrings(nodes)
	for i, n := range nodes {
		id := fmt.Sprintf("n%d", i)
		nodeIDs[n] = id
		b.WriteString(fmt.Sprintf("    %s[\"%s\"]\n", id, filepath.Base(n)))
	}

	for _, e := range delta.edgesAdded {
		fromID := nodeIDs[e.from]
		if fromID == "" {
			fromID = fmt.Sprintf("anon_%d", len(nodeIDs))
			b.WriteString(fmt.Sprintf("    %s[\"%s\"]\n", fromID, filepath.Base(e.from)))
			nodeIDs[e.from] = fromID
		}
		toID := nodeIDs[e.to]
		if toID == "" {
			toID = fmt.Sprintf("anon_%d", len(nodeIDs))
			b.WriteString(fmt.Sprintf("    %s[\"%s\"]\n", toID, filepath.Base(e.to)))
			nodeIDs[e.to] = toID
		}
		b.WriteString(fmt.Sprintf("    %s --> %s\n", fromID, toID))
	}
	for _, e := range delta.edgesRemoved {
		fromID := nodeIDs[e.from]
		if fromID == "" {
			fromID = fmt.Sprintf("anon_%d", len(nodeIDs))
			b.WriteString(fmt.Sprintf("    %s[\"%s\"]\n", fromID, filepath.Base(e.from)))
			nodeIDs[e.from] = fromID
		}
		toID := nodeIDs[e.to]
		if toID == "" {
			toID = fmt.Sprintf("anon_%d", len(nodeIDs))
			b.WriteString(fmt.Sprintf("    %s[\"%s\"]\n", toID, filepath.Base(e.to)))
			nodeIDs[e.to] = toID
		}
		b.WriteString(fmt.Sprintf("    %s -.-> %s\n", fromID, toID))
	}

	if len(delta.changedNodes) > 0 {
		addedClasses := make([]string, 0, len(delta.changedNodes))
		for _, n := range sortedChangedNodes(delta.changedNodes) {
			if id := nodeIDs[n]; id != "" {
				addedClasses = append(addedClasses, id)
			}
		}
		if len(addedClasses) > 0 {
			b.WriteString("    classDef added fill:#d9f2d9,stroke:#2e8b57,color:#000000\n")
			b.WriteString(fmt.Sprintf("    class %s added\n", strings.Join(addedClasses, ",")))
		}
	}
	if len(delta.nodesRemoved) > 0 {
		removedClasses := make([]string, 0, len(delta.nodesRemoved))
		for _, n := range delta.nodesRemoved {
			if id := nodeIDs[n]; id != "" {
				removedClasses = append(removedClasses, id)
			}
		}
		if len(removedClasses) > 0 {
			b.WriteString("    classDef removed fill:#f8d7da,stroke:#b22222,color:#000000\n")
			b.WriteString(fmt.Sprintf("    class %s removed\n", strings.Join(removedClasses, ",")))
		}
	}

	unchangedClasses := make([]string, 0, len(nodeIDs))
	for path, id := range nodeIDs {
		if _, changed := delta.changedNodes[path]; changed {
			continue
		}
		unchangedClasses = append(unchangedClasses, id)
	}
	sort.Strings(unchangedClasses)
	if len(unchangedClasses) > 0 {
		b.WriteString("    classDef unchanged fill:#f5f6f8,stroke:#c3c7cf,color:#667085,stroke-dasharray: 5 3\n")
		b.WriteString(fmt.Sprintf("    class %s unchanged\n", strings.Join(unchangedClasses, ",")))
	}

	return b.String()
}

func dedupeSortedStrings(values []string) []string {
	if len(values) == 0 {
		return values
	}
	sort.Strings(values)
	result := make([]string, 0, len(values))
	prev := ""
	for i, value := range values {
		if i == 0 || value != prev {
			result = append(result, value)
			prev = value
		}
	}
	return result
}
