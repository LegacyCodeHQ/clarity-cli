package json

import (
	"encoding/json"

	"github.com/LegacyCodeHQ/sanity/cmd/graph/formatters/common"
	"github.com/LegacyCodeHQ/sanity/parsers"
)

// JSONFormatter formats dependency graphs as JSON.
type JSONFormatter struct{}

// Format converts the dependency graph to JSON format.
// The opts parameter is accepted for interface compatibility but not used.
func (f *JSONFormatter) Format(g parsers.DependencyGraph, opts common.FormatOptions) (string, error) {
	data, err := json.MarshalIndent(g, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// GenerateURL returns false as JSON format does not support URL generation.
func (f *JSONFormatter) GenerateURL(output string) (string, bool) {
	return "", false
}
