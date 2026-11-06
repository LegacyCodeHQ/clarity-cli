package lean_test

import (
	"context"
	"testing"

	"github.com/LegacyCodeHQ/clarity/internal/tree_sitter/lean"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/stretchr/testify/require"
)

func TestCanParseLeanSource(t *testing.T) {
	parser := sitter.NewParser()
	defer parser.Close()
	parser.SetLanguage(lean.GetLanguage())
	tree, err := parser.ParseCtx(context.Background(), nil, []byte("import MyLib.Basic\ntheorem identity (n : Nat) : n = n := by\n  rfl\n"))
	require.NoError(t, err)
	defer tree.Close()
	require.False(t, tree.RootNode().HasError(), tree.RootNode().String())
	require.Equal(t, "module", tree.RootNode().Type())
}
