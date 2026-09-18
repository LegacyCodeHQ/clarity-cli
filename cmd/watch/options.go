package watch

import "github.com/LegacyCodeHQ/clarity/cmd/show/formatters"

type watchOptions struct {
	worktreePath string
	port         int
	direction    string
	format       string
	includeExt   string
	excludeExt   string
	includes     []string
	excludes     []string
	betweenFiles []string
	moduleSelect string
	reach        string
	depthLevel   int
	pruneFiles   []string
	all          bool
	collapse     bool
	edgeLabels   bool
	noStats      bool
	noPhantom    bool
	dbPath       string
}

func defaultWatchOptions() *watchOptions {
	return &watchOptions{
		port:       4900,
		direction:  formatters.DefaultDirection.StringLower(),
		format:     formatters.OutputFormatDOT.String(),
		depthLevel: 1,
	}
}
