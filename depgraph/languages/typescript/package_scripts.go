package typescript

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/LegacyCodeHQ/clarity/vcs"
)

// ResolvePackageJSONScriptPaths extracts shell-script references from a
// package.json's "scripts" field and returns the supplied project paths
// they resolve to.
//
// Returns (nil, nil) for any file that isn't literally named package.json —
// callers route every ".json" file through here, since npm-workspace
// discovery (npm_workspace.go) already established that as the only JSON
// file this provider understands.
func ResolvePackageJSONScriptPaths(
	absPath string,
	suppliedFiles map[string]bool,
	contentReader vcs.ContentReader,
) ([]string, error) {
	if filepath.Base(absPath) != "package.json" {
		return nil, nil
	}

	content, err := contentReader(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", absPath, err)
	}

	scripts := ParsePackageJSONScripts(content)
	if len(scripts) == 0 {
		return nil, nil
	}

	sourceDir := filepath.Dir(absPath)
	var projectImports []string
	seen := make(map[string]bool)
	for _, command := range scripts {
		for _, ref := range ShellScriptReferences(command) {
			resolved := filepath.Clean(filepath.Join(sourceDir, ref))
			if !suppliedFiles[resolved] || seen[resolved] {
				continue
			}
			seen[resolved] = true
			projectImports = append(projectImports, resolved)
		}
	}
	return projectImports, nil
}

// ParsePackageJSONScripts extracts the "scripts" field of a package.json as
// script name -> command line. Returns nil if the field is missing or the
// document is malformed.
func ParsePackageJSONScripts(data []byte) map[string]string {
	var raw struct {
		Scripts map[string]string `json:"scripts"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}
	return raw.Scripts
}

// ShellScriptReferences splits an npm script command line on shell word
// boundaries and returns the tokens that look like a reference to a .sh
// file (a relative or bare path ending in ".sh"), in first-seen order with
// surrounding quotes and glued shell operators ("&&", ";", "|") stripped.
//
// This is a lightweight token scan, not a shell parser: it is meant to
// catch the common "./scripts/foo.sh" / "bash scripts/foo.sh" shapes, not
// to fully understand arbitrary shell syntax (subshells, variable
// expansion, heredocs).
func ShellScriptReferences(command string) []string {
	var refs []string
	for _, field := range strings.Fields(command) {
		field = strings.Trim(field, `'"`)
		field = strings.TrimFunc(field, func(r rune) bool {
			return strings.ContainsRune("&;|()", r)
		})
		if field == "" || strings.HasPrefix(field, "-") || !strings.HasSuffix(field, ".sh") {
			continue
		}
		refs = append(refs, field)
	}
	return refs
}
