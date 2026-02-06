package swift

import (
	"context"
	"fmt"
	"os"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/swift"
)

// SwiftImport represents an import in a Swift file.
type SwiftImport struct {
	Path string
}

// SwiftImports parses a Swift file and returns its imports.
func SwiftImports(filePath string) ([]SwiftImport, error) {
	sourceCode, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return ParseSwiftImports(sourceCode)
}

// ParseSwiftImports parses Swift source code and extracts imports.
func ParseSwiftImports(sourceCode []byte) ([]SwiftImport, error) {
	lang := swift.GetLanguage()

	parser := sitter.NewParser()
	parser.SetLanguage(lang)

	tree, err := parser.ParseCtx(context.Background(), nil, sourceCode)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Swift code: %w", err)
	}
	defer tree.Close()

	return extractImports(tree.RootNode(), sourceCode), nil
}

func extractImports(rootNode *sitter.Node, sourceCode []byte) []SwiftImport {
	var imports []SwiftImport

	var walk func(*sitter.Node)
	walk = func(n *sitter.Node) {
		if n == nil {
			return
		}

		if n.Type() == "import_declaration" {
			if module := extractImportModule(n, sourceCode); module != "" {
				imports = append(imports, SwiftImport{Path: module})
			}
		}

		for i := 0; i < int(n.ChildCount()); i++ {
			walk(n.Child(i))
		}
	}

	walk(rootNode)
	return imports
}

func extractImportModule(node *sitter.Node, sourceCode []byte) string {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}
		if child.Type() == "identifier" {
			return strings.TrimSpace(child.Content(sourceCode))
		}
	}
	return ""
}
