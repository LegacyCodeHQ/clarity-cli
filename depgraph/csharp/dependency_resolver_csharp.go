package csharp

import (
	"fmt"

	"github.com/LegacyCodeHQ/sanity/vcs"
)

func ResolveCSharpProjectImports(
	absPath string,
	filePath string,
	suppliedFiles map[string]bool,
	contentReader vcs.ContentReader,
) ([]string, error) {
	content, err := contentReader(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", absPath, err)
	}

	_ = ParseCSharpImports(string(content))
	return []string{}, nil
}
