package quint

import (
	"path/filepath"
	"strings"

	"github.com/LegacyCodeHQ/clarity/vcs"
)

func IsTestFile(filePath string, contentReader vcs.ContentReader) bool {
	if filepath.Ext(filePath) != ".qnt" {
		return false
	}
	stem := strings.TrimSuffix(filepath.Base(filePath), ".qnt")
	path := "/" + filepath.ToSlash(filePath)
	if strings.HasSuffix(stem, "Test") || strings.HasSuffix(stem, "Tests") ||
		strings.HasSuffix(stem, "_test") || strings.HasSuffix(stem, "_tests") ||
		strings.HasPrefix(stem, "test_") || strings.Contains(path, "/test/") || strings.Contains(path, "/tests/") {
		return true
	}
	if contentReader == nil {
		return false
	}
	content, err := contentReader(filePath)
	if err != nil {
		return false
	}
	info, err := parseSource(content)
	return err == nil && info.hasTests
}
