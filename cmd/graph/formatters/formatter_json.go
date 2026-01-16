package formatters

import (
	"encoding/json"

	"github.com/LegacyCodeHQ/sanity/parsers"
)

// ToJSON converts the dependency graph to JSON format
func ToJSON(g parsers.DependencyGraph) ([]byte, error) {
	return json.MarshalIndent(g, "", "  ")
}
