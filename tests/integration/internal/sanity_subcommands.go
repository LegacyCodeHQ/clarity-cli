package internal

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func GraphSubcommand(t *testing.T, commit string) string {
	t.Helper()

	repoRoot := repoRoot(t)
	cmd := exec.Command("go", "run", "./main.go", "graph", "-c", commit, "-f", "dot")
	cmd.Dir = repoRoot
	cmd.Env = os.Environ()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	require.NoError(t, err, "stderr: %s", strings.TrimSpace(stderr.String()))

	return strings.TrimRight(stdout.String(), "\n")
}

func repoRoot(t *testing.T) string {
	t.Helper()

	wd, err := os.Getwd()
	require.NoError(t, err)

	repoRoot := filepath.Clean(filepath.Join(wd, "..", "..", ".."))
	_, err = os.Stat(filepath.Join(repoRoot, "go.mod"))
	require.NoError(t, err, "expected repo root with go.mod, got %s", repoRoot)

	return repoRoot
}
