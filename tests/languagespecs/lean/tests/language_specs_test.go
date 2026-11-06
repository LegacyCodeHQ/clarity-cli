package graph_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/LegacyCodeHQ/clarity/internal/testhelpers"
	"github.com/LegacyCodeHQ/clarity/tests/internal"
	"github.com/stretchr/testify/require"
)

func TestGraphLanguageSpec_LeanBasic(t *testing.T) {
	repoRoot := internal.RepoRoot(t)
	inputPath := filepath.Join(repoRoot, "tests/languagespecs/lean/examples/basic")
	output := internal.GraphSubcommandInputWithRepo(t, t.TempDir(), inputPath)
	testhelpers.DotGoldie(t).Assert(t, t.Name(), []byte(output))
}

func TestGraphLeanLakeTestDriver(t *testing.T) {
	project := t.TempDir()
	files := map[string]string{
		"lakefile.toml":      "name = 'dxn'\ntestDriver = 'DxnTest'\n[[lean_lib]]\nname = 'Dxn'\n[[lean_lib]]\nname = 'DxnTest'\n",
		"Dxn.lean":           "def value := 1\n",
		"DxnTest.lean":       "import DxnTest.Basic\n",
		"DxnTest/Basic.lean": "import Dxn\nexample : value = 1 := rfl\n",
		"DxnTest/Dxn.lean":   "import Dxn\n",
	}
	for name, content := range files {
		path := filepath.Join(project, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	}
	output := internal.GraphSubcommandInputWithRepo(t, project, project)
	require.Contains(t, output, `"DxnTest.lean" [label="DxnTest.lean", style=filled, fillcolor=lightgreen]`)
	require.Contains(t, output, `"DxnTest/Basic.lean" [label="Basic.lean", style=filled, fillcolor=lightgreen]`)
	require.Regexp(t, `(?m)^  "Dxn\.lean" \[label="[^"]+", style=filled, fillcolor=white\];$`, output)
	require.Contains(t, output, `"DxnTest/Basic.lean" -> "Dxn.lean"`)
	require.Contains(t, output, `"DxnTest/Dxn.lean" -> "Dxn.lean"`)
	require.NotContains(t, output, `"DxnTest/Basic.lean" -> "DxnTest/Dxn.lean"`)
}
