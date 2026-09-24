package typescript

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePackageJSONScripts_Basic(t *testing.T) {
	scripts := ParsePackageJSONScripts([]byte(`{
  "name": "app",
  "scripts": {
    "build": "./scripts/build.sh",
    "test": "vitest run"
  }
}`))
	assert.Equal(t, map[string]string{
		"build": "./scripts/build.sh",
		"test":  "vitest run",
	}, scripts)
}

func TestParsePackageJSONScripts_NoScriptsField(t *testing.T) {
	assert.Empty(t, ParsePackageJSONScripts([]byte(`{"name": "app"}`)))
}

func TestParsePackageJSONScripts_Malformed(t *testing.T) {
	assert.Empty(t, ParsePackageJSONScripts([]byte(`not json`)))
}

func TestShellScriptReferences(t *testing.T) {
	tests := []struct {
		name    string
		command string
		want    []string
	}{
		{"leading dot-slash", "./scripts/build.sh", []string{"./scripts/build.sh"}},
		{"bash prefix", "bash scripts/deploy.sh", []string{"scripts/deploy.sh"}},
		{"sh prefix", "sh scripts/deploy.sh", []string{"scripts/deploy.sh"}},
		{"chained commands", "chmod +x ./scripts/deploy.sh && ./scripts/deploy.sh --prod", []string{"./scripts/deploy.sh", "./scripts/deploy.sh"}},
		{"no shell script", "vitest run", nil},
		{"flag is not a path", "--config=deploy.sh", nil},
		{"quoted path", `"./scripts/build.sh"`, []string{"./scripts/build.sh"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ShellScriptReferences(tt.command))
		})
	}
}

func TestResolvePackageJSONScriptPaths(t *testing.T) {
	dir := t.TempDir()
	pkgJSON := filepath.Join(dir, "package.json")
	scriptDir := filepath.Join(dir, "scripts")
	require.NoError(t, os.MkdirAll(scriptDir, 0o755))
	buildSh := filepath.Join(scriptDir, "build.sh")
	require.NoError(t, os.WriteFile(buildSh, []byte("#!/bin/sh\necho build\n"), 0o644))
	require.NoError(t, os.WriteFile(pkgJSON, []byte(`{
  "name": "app",
  "scripts": {
    "build": "./scripts/build.sh",
    "test": "vitest run"
  }
}`), 0o644))

	suppliedFiles := map[string]bool{pkgJSON: true, buildSh: true}
	reader := func(path string) ([]byte, error) { return os.ReadFile(path) }

	resolved, err := ResolvePackageJSONScriptPaths(pkgJSON, suppliedFiles, reader)
	require.NoError(t, err)
	assert.Equal(t, []string{buildSh}, resolved)
}

func TestResolvePackageJSONScriptPaths_NotAPackageJSON(t *testing.T) {
	dir := t.TempDir()
	other := filepath.Join(dir, "tsconfig.json")
	require.NoError(t, os.WriteFile(other, []byte(`{"scripts": {"build": "./build.sh"}}`), 0o644))

	reader := func(path string) ([]byte, error) { return os.ReadFile(path) }
	resolved, err := ResolvePackageJSONScriptPaths(other, map[string]bool{other: true}, reader)
	require.NoError(t, err)
	assert.Empty(t, resolved)
}

func TestResolvePackageJSONScriptPaths_ScriptNotSupplied(t *testing.T) {
	dir := t.TempDir()
	pkgJSON := filepath.Join(dir, "package.json")
	require.NoError(t, os.WriteFile(pkgJSON, []byte(`{
  "scripts": {"build": "./scripts/build.sh"}
}`), 0o644))

	reader := func(path string) ([]byte, error) { return os.ReadFile(path) }
	resolved, err := ResolvePackageJSONScriptPaths(pkgJSON, map[string]bool{pkgJSON: true}, reader)
	require.NoError(t, err)
	assert.Empty(t, resolved)
}
