package diff

type graphEdge struct {
	from string
	to   string
}

type graphDelta struct {
	nodesAdded   []string
	nodesRemoved []string
	edgesAdded   []graphEdge
	edgesRemoved []graphEdge
	findings     []string
	changedNodes map[string]struct{}
}
