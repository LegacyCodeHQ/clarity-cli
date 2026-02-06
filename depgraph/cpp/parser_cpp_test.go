package cpp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCppIncludes(t *testing.T) {
	source := `
#include <vector>
#include "foo.hpp"
#include "utils"
`
	includes, err := ParseCppIncludes([]byte(source))

	require.NoError(t, err)
	assert.Len(t, includes, 3)

	assert.Equal(t, IncludeSystem, includes[0].Kind)
	assert.Equal(t, "vector", includes[0].Path)

	assert.Equal(t, IncludeLocal, includes[1].Kind)
	assert.Equal(t, "foo.hpp", includes[1].Path)
	assert.Equal(t, IncludeLocal, includes[2].Kind)
	assert.Equal(t, "utils", includes[2].Path)
}

func TestCppIncludes_ValidFile(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "main.cpp")

	content := `
#include "lib.hpp"
`
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	require.NoError(t, err)

	includes, err := CppIncludes(tmpFile)

	require.NoError(t, err)
	assert.Len(t, includes, 1)
	assert.Equal(t, "lib.hpp", includes[0].Path)
}

func TestResolveCppIncludePath(t *testing.T) {
	suppliedFiles := map[string]bool{
		"/project/include/lib.hpp": true,
		"/project/src/utils.h":     true,
		"/project/src/tools.hh":    true,
	}

	sourceFile := "/project/src/main.cpp"

	resolved := ResolveCppIncludePath(sourceFile, "../include/lib.hpp", suppliedFiles)
	assert.Contains(t, resolved, "/project/include/lib.hpp")

	resolved = ResolveCppIncludePath(sourceFile, "utils", suppliedFiles)
	assert.Contains(t, resolved, "/project/src/utils.h")

	resolved = ResolveCppIncludePath(sourceFile, "tools", suppliedFiles)
	assert.Contains(t, resolved, "/project/src/tools.hh")
}
