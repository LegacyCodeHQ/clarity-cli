package kotlin

import (
	"path/filepath"
	"strings"
)

// IsTestFile reports whether the given Kotlin file path is a test file.
func IsTestFile(filePath string) bool {
	base := filepath.Base(filePath)
	ext := filepath.Ext(base)
	if ext != ".kt" && ext != ".kts" {
		return false
	}

	if strings.HasSuffix(base, "Test"+ext) || strings.HasSuffix(base, "Tests"+ext) {
		return true
	}

	slashed := filepath.ToSlash(filePath)
	return strings.Contains(slashed, "/src/test/") || strings.Contains(slashed, "/test/")
}
