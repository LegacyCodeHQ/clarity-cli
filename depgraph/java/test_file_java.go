package java

import (
	"path/filepath"
	"strings"
)

// IsTestFile reports whether the given Java file path is a test file.
func IsTestFile(filePath string) bool {
	base := filepath.Base(filePath)
	if filepath.Ext(base) != ".java" {
		return false
	}

	if strings.HasSuffix(base, "Test.java") || strings.HasSuffix(base, "Tests.java") {
		return true
	}

	slashed := filepath.ToSlash(filePath)
	return strings.Contains(slashed, "/src/test/") || strings.Contains(slashed, "/test/")
}
