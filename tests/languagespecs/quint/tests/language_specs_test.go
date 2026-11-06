package graph_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/LegacyCodeHQ/clarity/tests/internal"
	"github.com/stretchr/testify/require"
)

func TestGraphQuintImports(t *testing.T) {
	project := t.TempDir()
	files := map[string]string{
		"counter.qnt": "module counter { const LIMIT: int\nvar count: int\naction init = count' = 0\n}\n",
		"facade.qnt":  "module facade { import counter(LIMIT = 3) as C from \"./counter\"\nexport C.*\n}\n",
		"checks.qnt":  "module checks { import facade.* from \"./facade\"\nrun initialTest = init\n}\n",
		"local.qnt":   "module local { pure val value = 1 }\nmodule sameFile { import local.* }\n",
	}
	for name, content := range files {
		require.NoError(t, os.WriteFile(filepath.Join(project, name), []byte(content), 0o644))
	}
	output := internal.GraphSubcommandInputWithRepo(t, project, project)
	require.Contains(t, output, `"facade.qnt" -> "counter.qnt"`)
	require.Contains(t, output, `"checks.qnt" -> "facade.qnt"`)
	require.Contains(t, output, `"checks.qnt" [label="checks.qnt", style=filled, fillcolor=lightgreen]`)
	require.NotContains(t, output, `"local.qnt" ->`)
}
