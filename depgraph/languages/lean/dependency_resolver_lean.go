package lean

import (
	"fmt"
	"path/filepath"

	"github.com/LegacyCodeHQ/clarity/vcs"
)

// ResolveLeanProjectImports resolves explicit imports against supplied source
// files. Lake packages use declared source roots; standalone files use ancestors
// of the importing file, nearest first. LEAN_PATH is not evaluated.
func ResolveLeanProjectImports(absPath string, suppliedFiles map[string]bool, contentReader vcs.ContentReader) ([]string, error) {
	return resolveLeanProjectImports(absPath, suppliedFiles, contentReader, func(path string) []string {
		return leanSourceRoots(path, contentReader)
	})
}

func resolveLeanProjectImports(absPath string, suppliedFiles map[string]bool, contentReader vcs.ContentReader, sourceRoots func(string) []string) ([]string, error) {
	content, err := contentReader(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", absPath, err)
	}
	imports, err := ParseImports(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse imports in %s: %w", absPath, err)
	}
	if len(imports) == 0 {
		return nil, nil
	}

	var dependencies []string
	seen := make(map[string]bool)
	roots := sourceRoots(absPath)
	for _, name := range imports {
		relativePath, ok := modulePath(name)
		if !ok {
			continue
		}
		for _, dir := range roots {
			candidate := filepath.Join(dir, relativePath)
			if suppliedFiles[candidate] {
				if candidate != absPath && !seen[candidate] {
					seen[candidate] = true
					dependencies = append(dependencies, candidate)
				}
				break
			}
		}
	}
	return dependencies, nil
}
