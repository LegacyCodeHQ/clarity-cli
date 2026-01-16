package parsers

import "encoding/json"

// ToJSON converts the dependency graph to JSON format
func (g DependencyGraph) ToJSON() ([]byte, error) {
	return json.MarshalIndent(g, "", "  ")
}
