package swift

import (
	"path/filepath"
	"strings"
)

// IsTestFile reports whether the given Swift path is a test file.
func IsTestFile(filePath string) bool {
	fileName := filepath.Base(filePath)
	ext := filepath.Ext(fileName)
	if ext != ".swift" {
		return false
	}

	base := strings.TrimSuffix(fileName, ext)
	if strings.HasSuffix(base, "Tests") || strings.HasSuffix(base, "Test") {
		return true
	}

	path := filepath.ToSlash(filePath)
	return strings.Contains(path, "/Tests/")
}
