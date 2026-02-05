package depgraph

import (
	"path/filepath"
	"testing"
)

func TestCollectDependencyGraphFiles_IncludesKotlinScriptFilesInKotlinIndex(t *testing.T) {
	tmpDir := t.TempDir()
	kotlinScriptPath := filepath.Join(tmpDir, "build.kts")

	_, _, _, kotlinFiles, _, err := collectDependencyGraphFiles([]string{kotlinScriptPath})
	if err != nil {
		t.Fatalf("collectDependencyGraphFiles() error = %v", err)
	}

	if len(kotlinFiles) != 1 {
		t.Fatalf("kotlinFiles count = %d, want 1", len(kotlinFiles))
	}
}
