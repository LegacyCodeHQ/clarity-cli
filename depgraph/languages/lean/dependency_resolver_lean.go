package lean

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/LegacyCodeHQ/clarity/vcs"
)

// ResolveLeanProjectImports resolves explicit imports against supplied source
// files. Ancestors of the importing file are candidate source roots, searched
// nearest first. Lake configuration and LEAN_PATH are not evaluated.
func ResolveLeanProjectImports(absPath string, suppliedFiles map[string]bool, contentReader vcs.ContentReader) ([]string, error) {
	content, err := contentReader(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", absPath, err)
	}
	imports, err := ParseImports(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse imports in %s: %w", absPath, err)
	}

	var dependencies []string
	seen := make(map[string]bool)
	for _, name := range imports {
		relativePath, ok := modulePath(name)
		if !ok {
			continue
		}
		for dir := filepath.Dir(absPath); ; dir = filepath.Dir(dir) {
			candidate := filepath.Join(dir, relativePath)
			if suppliedFiles[candidate] {
				if candidate != absPath && !seen[candidate] {
					seen[candidate] = true
					dependencies = append(dependencies, candidate)
				}
				break
			}
			if filepath.Dir(dir) == dir {
				break
			}
		}
	}
	return dependencies, nil
}

// Dots separate name components unless enclosed in Lean's identifier quotes.
func modulePath(name string) (string, bool) {
	var component strings.Builder
	var components []string
	quoted := false
	for _, r := range name {
		switch {
		case r == '«':
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
