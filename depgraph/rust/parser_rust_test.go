package rust

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRustImports(t *testing.T) {
	source := `
use std::io;
use crate::utils::helper as h;
extern crate serde;
mod nested;
`
	imports, err := ParseRustImports([]byte(source))

	require.NoError(t, err)
	assert.Len(t, imports, 4)
	assert.Equal(t, "std::io", imports[0].Path)
	assert.Equal(t, RustImportUse, imports[0].Kind)
	assert.Equal(t, "crate::utils::helper", imports[1].Path)
	assert.Equal(t, RustImportUse, imports[1].Kind)
	assert.Equal(t, "serde", imports[2].Path)
	assert.Equal(t, RustImportExternCrate, imports[2].Kind)
	assert.Equal(t, "nested", imports[3].Path)
	assert.Equal(t, RustImportModDecl, imports[3].Kind)
}

func TestRustImports_ValidFile(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "lib.rs")

	content := `
use std::fmt;
`
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	require.NoError(t, err)

	imports, err := RustImports(tmpFile)

	require.NoError(t, err)
	assert.Len(t, imports, 1)
	assert.Equal(t, "std::fmt", imports[0].Path)
	assert.Equal(t, RustImportUse, imports[0].Kind)
}
