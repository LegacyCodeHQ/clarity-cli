package quint

import (
	"fmt"
	"path/filepath"

	"github.com/LegacyCodeHQ/clarity/vcs"
)

// ResolveQuintProjectImports resolves only explicit file imports. Imports without
// from refer to modules already loaded by Quint, not implicit sibling files.
func ResolveQuintProjectImports(absPath string, suppliedFiles map[string]bool, contentReader vcs.ContentReader) ([]string, error) {
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
	for _, source := range imports {
		// Quint 0.33 appends .qnt unconditionally, even if source already has
		// an extension. Use the supplied snapshot, not filesystem existence.
		path := filepath.Join(filepath.Dir(absPath), filepath.FromSlash(source)+".qnt")
		if path != absPath && suppliedFiles[path] && !seen[path] {
			seen[path] = true
			dependencies = append(dependencies, path)
		}
	}
	return dependencies, nil
}
