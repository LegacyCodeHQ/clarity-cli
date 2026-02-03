package common

import "github.com/LegacyCodeHQ/sanity/vcs"

// FormatOptions contains optional parameters for formatting dependency graphs.
type FormatOptions struct {
	// Label is an optional title or label for the graph
	Label string
	// FileStats contains file statistics (additions/deletions) for display in nodes
	FileStats map[string]vcs.FileStats
}
