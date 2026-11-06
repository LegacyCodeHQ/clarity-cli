package lean

import (
	"context"
	"fmt"
	"sync"

	tree_sitter_lean "github.com/LegacyCodeHQ/clarity/internal/tree_sitter/lean"
	sitter "github.com/smacker/go-tree-sitter"
)

var leanParserPool = sync.Pool{
	New: func() any {
		parser := sitter.NewParser()
		parser.SetLanguage(tree_sitter_lean.GetLanguage())
		return parser
	},
}

// ParseImports extracts explicit module imports, preserving source order.
// Syntax errors in declarations do not discard the file's import header.
func ParseImports(sourceCode []byte) ([]string, error) {
	parser, _ := leanParserPool.Get().(*sitter.Parser)
	if parser == nil {
		return nil, fmt.Errorf("failed to acquire Lean parser")
	}
	defer leanParserPool.Put(parser)

	tree, err := parser.ParseCtx(context.Background(), nil, sourceCode)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Lean code: %w", err)
	}
	defer tree.Close()

	var imports []string
	seen := make(map[string]bool)
	root := tree.RootNode()
	for i := 0; i < int(root.NamedChildCount()); i++ {
		node := root.NamedChild(i)
		if node.Type() != "import" {
			continue
		}
		name := node.ChildByFieldName("name")
		if name == nil || name.IsMissing() {
			continue
		}
		path := name.Content(sourceCode)
		if !seen[path] {
			seen[path] = true
			imports = append(imports, path)
		}
	}
	return imports, nil
}
