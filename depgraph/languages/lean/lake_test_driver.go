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
	for dir := filepath.Dir(absPath); ; dir = filepath.Dir(dir) {
		if content, err := contentReader(filepath.Join(dir, "lakefile.toml")); err == nil {
			var config lakeConfig
			if toml.Unmarshal(content, &config) != nil || config.TestDriver == "" {
				return false
			}
			return config.isTestFile(dir, absPath)
		}
		// A nested Lean-configured package must not inherit its parent's driver.
		if _, err := contentReader(filepath.Join(dir, "lakefile.lean")); err == nil {
			return false
		}
		if filepath.Dir(dir) == dir {
			return false
		}
	}
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
