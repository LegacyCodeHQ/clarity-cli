package quint

import (
	"bytes"
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

var quintLanguage = tree_sitter_quint.GetLanguage()
var quintImportQuery, quintTestQuery *sitter.Query

func init() {
	var err error
	quintImportQuery, err = sitter.NewQuery([]byte(`
[
  (named_import_declaration source: (string_literal) @source)
  (wildcard_import_declaration source: (string_literal) @source)
  (module_import_declaration source: (string_literal) @source)
  (instance_declaration source: (string_literal) @source)
  (anonymous_instance_declaration source: (string_literal) @source)
]`), quintLanguage)
	if err != nil {
		panic("failed to compile Quint import query: " + err.Error())
	}
	quintTestQuery, err = sitter.NewQuery([]byte(`
[
  (operator_definition name: (_) @name)
  (value_definition name: (_) @name)
]`), quintLanguage)
	if err != nil {
		panic("failed to compile Quint test query: " + err.Error())
	}
}

var quintParserPool = sync.Pool{
	New: func() any {
		parser := sitter.NewParser()
		parser.SetLanguage(quintLanguage)
		return parser
	},
}

// ParseImports extracts explicit from clauses, including module instances.
// Quint string contents are literal, so backslashes are not unescaped.
func ParseImports(sourceCode []byte) ([]string, error) {
	info, err := parseSource(sourceCode, false)
	return info.imports, err
}

func parseSource(sourceCode []byte, tests bool) (sourceInfo, error) {
	query := quintImportQuery
	marker := []byte("from")
	if tests {
		query = quintTestQuery
		marker = []byte("Test")
	}
	// These are necessary literal tokens, not a replacement for syntax parsing.
	// A positive match still goes through tree-sitter to exclude comments/strings.
	if !bytes.Contains(sourceCode, marker) {
		return sourceInfo{}, nil
	}
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
	cursor := sitter.NewQueryCursor()
	defer cursor.Close()
	root := tree.RootNode()
	cursor.Exec(query, root)
	for {
		match, ok := cursor.NextMatch()
		if !ok {
			break
		}
		node := match.Captures[0].Node
		if node.IsMissing() {
			continue
		}
		declaration := node.Parent()
		if declaration == nil {
			continue
		}
		module := declaration.Parent()
		if module == nil || module.Type() != "module_definition" {
			continue
		}
		parent := module.Parent()
		if parent == nil || !parent.Equal(root) {
			continue
		}
		if tests {
			if strings.HasSuffix(node.Content(sourceCode), "Test") {
				info.hasTests = true
				return info, nil
			}
			continue
		}
		if node.HasError() {
			continue
		}
		literal := node.Content(sourceCode)
		if len(literal) < 3 || literal[0] != '"' || literal[len(literal)-1] != '"' {
			continue
		}
		path := literal[1 : len(literal)-1]
		if !seen[path] {
			seen[path] = true
			info.imports = append(info.imports, path)
		}
	}
	return info, nil
}
