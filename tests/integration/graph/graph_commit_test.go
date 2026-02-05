package graph_test

import (
	"testing"

	"github.com/LegacyCodeHQ/sanity/internal/testhelpers"
	"github.com/LegacyCodeHQ/sanity/tests/integration/internal"
)

func TestGraphCommit_AddOnboardCommand(t *testing.T) {
	output := internal.GraphSubcommand(t, "d2c296558f5a5261629fd43eff6b572c636bb3e8")

	g := testhelpers.DotGoldie(t)
	g.Assert(t, t.Name(), []byte(output))
}

func TestGraphCommit_AddInitAndPrimeCommands(t *testing.T) {
	output := internal.GraphSubcommand(t, "ba7ca46722c59ede139f329a3f1ef65c47b66832")

	g := testhelpers.DotGoldie(t)
	g.Assert(t, t.Name(), []byte(output))
}
