package javascript

import (
	"path/filepath"
	"strings"
)

// IsTestFile reports whether the given JavaScript/JSX/MJS path is a test file.
func IsTestFile(filePath string) bool {
	fileName := filepath.Base(filePath)
	ext := filepath.Ext(fileName)
	if ext != ".js" && ext != ".jsx" && ext != ".mjs" {
		return false
	}

	if strings.HasSuffix(fileName, ".test"+ext) || strings.HasSuffix(fileName, ".spec"+ext) {
		return true
	}

	return strings.Contains(filepath.ToSlash(filePath), "/__tests__/")
}
