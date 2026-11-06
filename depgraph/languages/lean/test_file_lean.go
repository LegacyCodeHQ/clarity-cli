package lean

import (
	"path/filepath"
	"strings"
)

// IsTestFile uses path conventions only: theorems and examples are ordinary
// Lean declarations and do not by themselves make a file a test.
func IsTestFile(filePath string) bool {
	if filepath.Ext(filePath) != ".lean" {
		return false
	}
	base := strings.ToLower(filepath.Base(filePath))
	path := "/" + strings.ToLower(filepath.ToSlash(filePath))
	return strings.HasSuffix(base, "_test.lean") || strings.HasPrefix(base, "test_") ||
		strings.Contains(path, "/test/") || strings.Contains(path, "/tests/")
}
