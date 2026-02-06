package csharp

import (
	"fmt"
	"os"
	"strings"
)

// CSharpImport represents a using directive.
type CSharpImport struct {
	Path string
}

// CSharpImports parses a C# file and returns its imports.
func CSharpImports(filePath string) ([]CSharpImport, error) {
	sourceCode, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return ParseCSharpImports(string(sourceCode)), nil
}

// ParseCSharpImports parses C# source code and extracts using directives.
func ParseCSharpImports(source string) []CSharpImport {
	lines := strings.Split(source, "\n")
	inBlockComment := false

	var imports []CSharpImport
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if inBlockComment {
			if end := strings.Index(trimmed, "*/"); end >= 0 {
				trimmed = strings.TrimSpace(trimmed[end+2:])
				inBlockComment = false
			} else {
				continue
			}
		}

		if start := strings.Index(trimmed, "/*"); start >= 0 {
			if end := strings.Index(trimmed[start+2:], "*/"); end >= 0 {
				trimmed = strings.TrimSpace(trimmed[:start] + trimmed[start+2+end+2:])
			} else {
				trimmed = strings.TrimSpace(trimmed[:start])
				inBlockComment = true
			}
		}

		if idx := strings.Index(trimmed, "//"); idx >= 0 {
			trimmed = strings.TrimSpace(trimmed[:idx])
		}

		if !strings.HasPrefix(trimmed, "using ") {
			continue
		}
		if !strings.Contains(trimmed, ";") {
			continue
		}

		statement := strings.TrimSpace(strings.TrimSuffix(trimmed, ";"))
		statement = strings.TrimPrefix(statement, "using ")
		statement = strings.TrimSpace(statement)

		if strings.HasPrefix(statement, "static ") {
			statement = strings.TrimSpace(strings.TrimPrefix(statement, "static "))
		}

		if eq := strings.Index(statement, "="); eq >= 0 {
			statement = strings.TrimSpace(statement[eq+1:])
		}

		if statement == "" || strings.HasPrefix(statement, "(") {
			continue
		}

		imports = append(imports, CSharpImport{Path: statement})
	}

	return imports
}
