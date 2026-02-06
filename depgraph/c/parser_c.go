package c

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/c"
)

// IncludeKind distinguishes between system and local includes.
type IncludeKind int

const (
	IncludeLocal IncludeKind = iota
	IncludeSystem
)

// Include represents a C include directive.
type Include struct {
	Path string
	Kind IncludeKind
}

// CIncludes parses a C file and returns its includes.
func CIncludes(filePath string) ([]Include, error) {
	sourceCode, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return ParseCIncludes(sourceCode)
}

// ParseCIncludes parses C source code and extracts includes.
func ParseCIncludes(sourceCode []byte) ([]Include, error) {
	lang := c.GetLanguage()

	parser := sitter.NewParser()
	parser.SetLanguage(lang)

	tree, err := parser.ParseCtx(context.Background(), nil, sourceCode)
	if err != nil {
		return nil, fmt.Errorf("failed to parse C code: %w", err)
	}
	defer tree.Close()

	return extractIncludes(tree.RootNode(), sourceCode), nil
}

func extractIncludes(rootNode *sitter.Node, sourceCode []byte) []Include {
	var includes []Include

	var walk func(*sitter.Node)
	walk = func(n *sitter.Node) {
		if n == nil {
			return
		}

		if n.Type() == "preproc_include" {
			if inc := extractIncludeFromNode(n, sourceCode); inc.Path != "" {
				includes = append(includes, inc)
			}
		}

		for i := 0; i < int(n.ChildCount()); i++ {
			walk(n.Child(i))
		}
	}

	walk(rootNode)
	return includes
}

func extractIncludeFromNode(node *sitter.Node, sourceCode []byte) Include {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}
		switch child.Type() {
		case "string_literal":
			return Include{Path: cleanStringLiteral(child.Content(sourceCode)), Kind: IncludeLocal}
		case "system_lib_string":
			return Include{Path: cleanSystemInclude(child.Content(sourceCode)), Kind: IncludeSystem}
		}
	}

	return Include{}
}

func cleanStringLiteral(raw string) string {
	return strings.Trim(raw, "\"' ")
}

func cleanSystemInclude(raw string) string {
	trimmed := strings.TrimSpace(raw)
	trimmed = strings.TrimPrefix(trimmed, "<")
	trimmed = strings.TrimSuffix(trimmed, ">")
	return strings.TrimSpace(trimmed)
}

// ResolveCIncludePath resolves a C include path to possible file paths.
func ResolveCIncludePath(sourceFile, includePath string, suppliedFiles map[string]bool) []string {
	sourceDir := filepath.Dir(sourceFile)
	basePath := filepath.Join(sourceDir, includePath)
	basePath = filepath.Clean(basePath)

	var resolvedPaths []string

	if filepath.Ext(includePath) != "" {
		if suppliedFiles[basePath] {
			resolvedPaths = append(resolvedPaths, basePath)
		}
		return resolvedPaths
	}

	candidate := basePath + ".h"
	if suppliedFiles[candidate] {
		resolvedPaths = append(resolvedPaths, candidate)
	}

	return resolvedPaths
}
