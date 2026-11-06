package quint_test

import (
	"context"
	"testing"

	"github.com/LegacyCodeHQ/clarity/internal/tree_sitter/quint"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/stretchr/testify/require"
)

func TestCanParseQuintSource(t *testing.T) {
	parser := sitter.NewParser()
	defer parser.Close()
	parser.SetLanguage(quint.GetLanguage())
	tree, err := parser.ParseCtx(context.Background(), nil, []byte("module main {\nimport Proto(X = 3).* from \"./proto\"\nrun initialTest = init\n}\n"))
	require.NoError(t, err)
	defer tree.Close()
	require.Equal(t, "source_file", tree.RootNode().Type())
	require.False(t, tree.RootNode().HasError(), tree.RootNode().String())
}
