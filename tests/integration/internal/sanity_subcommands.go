package internal

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	graphcmd "github.com/LegacyCodeHQ/sanity/cmd/graph"
	"github.com/stretchr/testify/require"
)

func GraphSubcommand(t *testing.T, commit string) string {
	t.Helper()

	repoRoot := repoRoot(t)
	cmd := graphcmd.NewCommand()
	cmd.SetArgs([]string{"-c", commit, "-f", "dot", "-r", repoRoot})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err := cmd.Execute()
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
