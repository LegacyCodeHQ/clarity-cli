package cpp

import (
	"path/filepath"
	"strings"
)

// IsTestFile reports whether the given C++ path is a test file.
func IsTestFile(filePath string) bool {
	fileName := filepath.Base(filePath)
	ext := filepath.Ext(fileName)
	supported := map[string]bool{
		".cc":  true,
		".cpp": true,
		".cxx": true,
		".hpp": true,
		".hh":  true,
		".hxx": true,
	}
	if !supported[ext] {
		return false
	}

	base := strings.TrimSuffix(fileName, ext)
	if strings.HasPrefix(base, "test_") || strings.HasSuffix(base, "_test") {
		return true
	}

	path := filepath.ToSlash(filePath)
	return strings.Contains(path, "/tests/") || strings.Contains(path, "/test/")
}
