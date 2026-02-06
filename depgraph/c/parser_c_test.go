package c

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCIncludes(t *testing.T) {
	source := `
#include <stdio.h>
#include "foo/bar.h"
#include "utils"
`
	includes, err := ParseCIncludes([]byte(source))

	require.NoError(t, err)
	assert.Len(t, includes, 3)

	assert.Equal(t, IncludeSystem, includes[0].Kind)
	assert.Equal(t, "stdio.h", includes[0].Path)

	assert.Equal(t, IncludeLocal, includes[1].Kind)
	assert.Equal(t, "foo/bar.h", includes[1].Path)
	assert.Equal(t, IncludeLocal, includes[2].Kind)
	assert.Equal(t, "utils", includes[2].Path)
}

func TestCIncludes_ValidFile(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "main.c")

	content := `
#include "lib.h"
`
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	require.NoError(t, err)

	includes, err := CIncludes(tmpFile)

	require.NoError(t, err)
	assert.Len(t, includes, 1)
	assert.Equal(t, "lib.h", includes[0].Path)
}

func TestResolveCIncludePath(t *testing.T) {
	suppliedFiles := map[string]bool{
		"/project/include/lib.h": true,
		"/project/src/utils.h":   true,
	}

	sourceFile := "/project/src/main.c"

	resolved := ResolveCIncludePath(sourceFile, "../include/lib.h", suppliedFiles)
	assert.Contains(t, resolved, "/project/include/lib.h")

	resolved = ResolveCIncludePath(sourceFile, "utils", suppliedFiles)
	assert.Contains(t, resolved, "/project/src/utils.h")
}
