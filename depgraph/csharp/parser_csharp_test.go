package csharp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCSharpImports(t *testing.T) {
	source := `
using System;
using System.Collections.Generic;
using static System.Math;
using Alias = MyApp.Core;

namespace App {
	// using NotAReal;
	using (var x = foo) { }
}
`
	imports := ParseCSharpImports(source)

	assert.Len(t, imports, 4)
	assert.Equal(t, "System", imports[0].Path)
	assert.Equal(t, "System.Collections.Generic", imports[1].Path)
	assert.Equal(t, "System.Math", imports[2].Path)
	assert.Equal(t, "MyApp.Core", imports[3].Path)
}

func TestCSharpImports_ValidFile(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "Program.cs")

	content := `
using System;
`
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	require.NoError(t, err)

	imports, err := CSharpImports(tmpFile)

	require.NoError(t, err)
	assert.Len(t, imports, 1)
	assert.Equal(t, "System", imports[0].Path)
}
