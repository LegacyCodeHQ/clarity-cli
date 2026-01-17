package formatters

import (
	"fmt"
	"sort"
	"strings"

	"github.com/LegacyCodeHQ/sanity/git"
	"github.com/LegacyCodeHQ/sanity/parsers"
)

// Format represents an output format type
type Format string

const (
	FormatDOT     Format = "dot"
	FormatJSON    Format = "json"
	FormatMermaid Format = "mermaid"
)

// String returns the string representation of the format
func (f Format) String() string {
	return string(f)
}

// FormatOptions contains optional parameters for formatting dependency graphs.
type FormatOptions struct {
	// Label is an optional title or label for the graph
	Label string
	// FileStats contains file statistics (additions/deletions) for display in nodes
	FileStats map[string]git.FileStats
}

// Formatter is the interface that all graph formatters must implement.
type Formatter interface {
	// Format converts a dependency graph to a formatted string representation.
	Format(g parsers.DependencyGraph, opts FormatOptions) (string, error)
}

// registry holds all registered formatters
var registry = make(map[Format]func() Formatter)

// Register adds a formatter constructor to the registry.
// Each formatter should call this in its init() function.
func Register(name Format, constructor func() Formatter) {
	registry[name] = constructor
}

// NewFormatter creates a Formatter for the specified format type.
func NewFormatter(format string) (Formatter, error) {
	constructor, ok := registry[Format(format)]
	if !ok {
		return nil, fmt.Errorf("unknown format: %s (valid options: %s)", format, availableFormats())
	}
	return constructor(), nil
}

// availableFormats returns a comma-separated list of registered format names.
func availableFormats() string {
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name.String())
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}
