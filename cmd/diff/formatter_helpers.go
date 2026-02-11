package diff

import "sort"

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
