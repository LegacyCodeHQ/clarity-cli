package lean

import (
	"path/filepath"
	"strings"

	"github.com/LegacyCodeHQ/clarity/vcs"
	"github.com/pelletier/go-toml/v2"
)

type lakeConfig struct {
	TestDriver  string       `toml:"testDriver"`
	SrcDir      string       `toml:"srcDir"`
	Libraries   []lakeTarget `toml:"lean_lib"`
	Executables []lakeTarget `toml:"lean_exe"`
}

type lakeTarget struct {
	Name   string   `toml:"name"`
	SrcDir string   `toml:"srcDir"`
	Roots  []string `toml:"roots"`
	Root   string   `toml:"root"`
}

// findLakePackage uses the source reader so package boundaries and configuration
// belong to the same snapshot as the importing file. A nonempty directory with
// nil config is a package whose configuration we cannot statically interpret.
func findLakePackage(absPath string, contentReader vcs.ContentReader) (string, *lakeConfig) {
	for dir := filepath.Dir(absPath); ; dir = filepath.Dir(dir) {
		// Lake prefers Lean configuration when both formats are present. We do
		// not evaluate it, and must not apply a dormant TOML configuration.
		if _, err := contentReader(filepath.Join(dir, "lakefile.lean")); err == nil {
			return dir, nil
		}
		if content, err := contentReader(filepath.Join(dir, "lakefile.toml")); err == nil {
			var config lakeConfig
			if toml.Unmarshal(content, &config) != nil {
				return dir, nil
			}
			return dir, &config
		}
		if filepath.Dir(dir) == dir {
			return "", nil
		}
	}
}

func leanSourceRoots(absPath string, contentReader vcs.ContentReader) []string {
	packageDir, config := findLakePackage(absPath, contentReader)
	if packageDir != "" {
		// Even an uninterpreted Lake configuration establishes a package boundary;
		// do not mistake arbitrary nested directories for Lean include paths.
		if config == nil {
			return []string{packageDir}
		}
		packageRoot := filepath.Join(packageDir, config.SrcDir)
		roots := []string{packageRoot}
		seen := map[string]bool{packageRoot: true}
		addTarget := func(target lakeTarget) {
			root := filepath.Join(packageRoot, target.SrcDir)
			if !seen[root] {
				seen[root] = true
				roots = append(roots, root)
			}
		}
		for _, target := range config.Libraries {
			addTarget(target)
		}
		for _, target := range config.Executables {
			addTarget(target)
		}
		return roots
	}

	// Standalone sources have no declared root; retain the ancestor heuristic.
	var roots []string
	for dir := filepath.Dir(absPath); ; dir = filepath.Dir(dir) {
		roots = append(roots, dir)
		if filepath.Dir(dir) == dir {
			return roots
		}
	}
}

// isLakeTestFile reads the nearest package's declarative configuration through
// the same reader as source files, so classification also works at Git revisions.
// Lean configuration files and script/dependency drivers are not evaluated.
func isLakeTestFile(filePath string, contentReader vcs.ContentReader) bool {
	if filepath.Ext(filePath) != ".lean" || contentReader == nil {
		return false
	}
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return false
	}
	packageDir, config := findLakePackage(absPath, contentReader)
	if config == nil || config.TestDriver == "" {
		return false
	}
	return config.isTestFile(packageDir, absPath)
}

func (c lakeConfig) isTestFile(packageDir, filePath string) bool {
	for _, target := range c.Libraries {
		if target.Name != c.TestDriver {
			continue
		}
		roots := target.Roots
		if roots == nil {
			roots = []string{target.Name}
		}
		for _, root := range roots {
			if matchesLakeRoot(packageDir, c.SrcDir, target.SrcDir, root, filePath, true) {
				return true
			}
		}
	}
	for _, target := range c.Executables {
		if target.Name != c.TestDriver {
			continue
		}
		root := target.Root
		if root == "" {
			root = target.Name
		}
		if matchesLakeRoot(packageDir, c.SrcDir, target.SrcDir, root, filePath, false) {
			return true
		}
	}
	return false
}

func matchesLakeRoot(packageDir, packageSrcDir, targetSrcDir, root, filePath string, submodules bool) bool {
	moduleFile, ok := modulePath(root)
	if !ok {
		return false
	}
	rootPath := filepath.Join(packageDir, packageSrcDir, targetSrcDir, moduleFile)
	if filePath == rootPath {
		return true
	}
	return submodules && strings.HasPrefix(filePath, strings.TrimSuffix(rootPath, ".lean")+string(filepath.Separator))
}
