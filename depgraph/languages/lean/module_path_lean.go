package lean

import (
	"path/filepath"
	"strings"
)

// Dots separate name components unless enclosed in Lean's identifier quotes.
func modulePath(name string) (string, bool) {
	var component strings.Builder
	var components []string
	quoted := false
	for _, r := range name {
		switch {
		case r == '«' && !quoted:
			quoted = true
		case r == '»':
			quoted = false
		case r == '.' && !quoted:
			components = append(components, component.String())
			component.Reset()
		default:
			component.WriteRune(r)
		}
	}
	components = append(components, component.String())
	for _, part := range components {
		if part == "" || part == "." || part == ".." || strings.ContainsAny(part, "/\\\x00") {
			return "", false
		}
	}
	return filepath.Join(components...) + ".lean", !quoted
}
