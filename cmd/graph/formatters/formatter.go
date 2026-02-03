package formatters

import (
	"fmt"

	"github.com/LegacyCodeHQ/sanity/cmd/graph/formatters/common"
	"github.com/LegacyCodeHQ/sanity/cmd/graph/formatters/dot"
	"github.com/LegacyCodeHQ/sanity/cmd/graph/formatters/json"
	"github.com/LegacyCodeHQ/sanity/cmd/graph/formatters/mermaid"
	"github.com/LegacyCodeHQ/sanity/parsers"
)

// FormatOptions is an alias for common.FormatOptions for backward compatibility.
type FormatOptions = common.FormatOptions

// Formatter is the interface that all graph formatters must implement.
type Formatter interface {
	// Format converts a dependency graph to a formatted string representation.
	Format(g parsers.DependencyGraph, opts common.FormatOptions) (string, error)
	// GenerateURL creates a shareable URL for the formatted output.
	// Returns the URL and true if supported, or ("", false) if not.
	GenerateURL(output string) (string, bool)
}

// GetExtensionColors re-exports common.GetExtensionColors for backward compatibility.
var GetExtensionColors = common.GetExtensionColors

// IsTestFile re-exports common.IsTestFile for backward compatibility.
var IsTestFile = common.IsTestFile

// NewFormatter creates a Formatter for the specified format type.
func NewFormatter(format string) (Formatter, error) {
	f, ok := ParseOutputFormat(format)
	if !ok {
		return nil, fmt.Errorf("unknown format: %s (valid options: dot, json, mermaid)", format)
	}

	switch f {
	case OutputFormatDOT:
		return &dot.DOTFormatter{}, nil
	case OutputFormatJSON:
		return &json.JSONFormatter{}, nil
	case OutputFormatMermaid:
		return &mermaid.MermaidFormatter{}, nil
	default:
		return nil, fmt.Errorf("unknown format: %s (valid options: dot, json, mermaid)", format)
	}
}
