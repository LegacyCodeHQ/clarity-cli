package graph

import (
	"fmt"
	"path/filepath"
)

// RawPath is a user-provided file path from CLI flags.
type RawPath string

// AbsolutePath is a normalized absolute filesystem path.
type AbsolutePath string

func (p AbsolutePath) String() string {
	return string(p)
}

// PathResolver resolves raw user paths relative to a configured base directory.
type PathResolver struct {
	baseDir AbsolutePath
}

func NewPathResolver(baseDir string) (PathResolver, error) {
	if baseDir == "" {
		baseDir = "."
	}

	absBaseDir, err := filepath.Abs(baseDir)
	if err != nil {
		return PathResolver{}, fmt.Errorf("failed to resolve base path: %w", err)
	}

	return PathResolver{baseDir: AbsolutePath(filepath.Clean(absBaseDir))}, nil
}

func (r PathResolver) Resolve(path RawPath) (AbsolutePath, error) {
	pathStr := string(path)
	if pathStr == "" {
		return "", fmt.Errorf("path cannot be empty")
	}

	if filepath.IsAbs(pathStr) {
		return AbsolutePath(filepath.Clean(pathStr)), nil
	}

	return AbsolutePath(filepath.Clean(filepath.Join(r.baseDir.String(), pathStr))), nil
}
