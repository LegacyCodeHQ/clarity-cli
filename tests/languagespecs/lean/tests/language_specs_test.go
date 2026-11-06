package graph_test

import (
	"path/filepath"
	"testing"

	"github.com/LegacyCodeHQ/clarity/internal/testhelpers"
	"github.com/LegacyCodeHQ/clarity/tests/internal"
)

func TestGraphLanguageSpec_LeanBasic(t *testing.T) {
	repoRoot := internal.RepoRoot(t)
	inputPath := filepath.Join(repoRoot, "tests/languagespecs/lean/examples/basic")
	output := internal.GraphSubcommandInputWithRepo(t, t.TempDir(), inputPath)
	testhelpers.DotGoldie(t).Assert(t, t.Name(), []byte(output))
}
