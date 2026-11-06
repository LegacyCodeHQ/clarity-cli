package quint

import (
	"context"
	"fmt"
	"strings"
	"sync"

	tree_sitter_quint "github.com/LegacyCodeHQ/clarity/internal/tree_sitter/quint"
	sitter "github.com/smacker/go-tree-sitter"
)

type sourceInfo struct {
	imports  []string
	hasTests bool
}

var quintParserPool = sync.Pool{
	New: func() any {
		parser := sitter.NewParser()
		parser.SetLanguage(tree_sitter_quint.GetLanguage())
		return parser
	},
}

// ParseImports extracts explicit from clauses, including module instances.
// Quint string contents are literal, so backslashes are not unescaped.
func ParseImports(sourceCode []byte) ([]string, error) {
	info, err := parseSource(sourceCode)
	return info.imports, err
}

func parseSource(sourceCode []byte) (sourceInfo, error) {
	parser, _ := quintParserPool.Get().(*sitter.Parser)
	if parser == nil {
		return sourceInfo{}, fmt.Errorf("failed to acquire Quint parser")
	}
	defer quintParserPool.Put(parser)
	tree, err := parser.ParseCtx(context.Background(), nil, sourceCode)
	if err != nil {
		return sourceInfo{}, fmt.Errorf("failed to parse Quint code: %w", err)
	}
	defer tree.Close()

	var info sourceInfo
	seen := make(map[string]bool)
	root := tree.RootNode()
	for i := 0; i < int(root.NamedChildCount()); i++ {
		module := root.NamedChild(i)
		if module.Type() != "module_definition" {
			continue
		}
		for j := 0; j < int(module.NamedChildCount()); j++ {
			node := module.NamedChild(j)
			switch node.Type() {
			case "named_import_declaration", "wildcard_import_declaration", "module_import_declaration",
				"instance_declaration", "anonymous_instance_declaration":
				source := node.ChildByFieldName("source")
				if source == nil || source.IsMissing() || source.HasError() {
					continue
				}
				literal := source.Content(sourceCode)
				if len(literal) < 3 || literal[0] != '"' || literal[len(literal)-1] != '"' {
					continue
				}
				path := literal[1 : len(literal)-1]
				if !seen[path] {
					seen[path] = true
					info.imports = append(info.imports, path)
				}
			case "operator_definition", "value_definition":
				name := node.ChildByFieldName("name")
				if name != nil && !name.IsMissing() && strings.HasSuffix(name.Content(sourceCode), "Test") {
					info.hasTests = true
				}
			}
		}
	}
	return info, nil
}
