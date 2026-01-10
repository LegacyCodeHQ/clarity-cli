package parser

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
)

// GoSymbolInfo tracks symbols defined and referenced in a Go file
type GoSymbolInfo struct {
	FilePath   string
	Package    string
	Defined    map[string]bool // Symbols defined in this file
	Referenced map[string]bool // Symbols referenced in this file
}

// ExtractGoSymbols analyzes a Go file and extracts defined and referenced symbols
func ExtractGoSymbols(filePath string) (*GoSymbolInfo, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	info := &GoSymbolInfo{
		FilePath:   filePath,
		Package:    node.Name.Name,
		Defined:    make(map[string]bool),
		Referenced: make(map[string]bool),
	}

	// Extract defined symbols (top-level declarations)
	for _, decl := range node.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			// Function or method
			if d.Recv == nil {
				// Top-level function
				info.Defined[d.Name.Name] = true
			} else {
				// Method - also track the receiver type
				if d.Name.IsExported() {
					info.Defined[d.Name.Name] = true
				}
			}
		case *ast.GenDecl:
			// Type, const, var, or import
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					info.Defined[s.Name.Name] = true
				case *ast.ValueSpec:
					for _, name := range s.Names {
						info.Defined[name.Name] = true
					}
				}
			}
		}
	}

	// Extract referenced symbols
	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.Ident:
			// Skip package names and defined symbols
			if x.Obj == nil && x.Name != "_" && x.Name != info.Package {
				info.Referenced[x.Name] = true
			}
		case *ast.SelectorExpr:
			// For qualified identifiers like fmt.Println, we only care about
			// package-local references, not external packages
			if ident, ok := x.X.(*ast.Ident); ok {
				// This is a selector like x.Field - track x
				if ident.Obj == nil {
					info.Referenced[ident.Name] = true
				}
			}
		}
		return true
	})

	return info, nil
}

// BuildIntraPackageDependencies builds dependencies between files in the same Go package
func BuildIntraPackageDependencies(filePaths []string) (map[string][]string, error) {
	// Group files by package
	packageFiles := make(map[string][]string)
	for _, filePath := range filePaths {
		if filepath.Ext(filePath) != ".go" {
			continue
		}

		absPath, err := filepath.Abs(filePath)
		if err != nil {
			continue
		}

		// Get package directory
		pkgDir := filepath.Dir(absPath)
		packageFiles[pkgDir] = append(packageFiles[pkgDir], absPath)
	}

	// Build dependencies for each package
	dependencies := make(map[string][]string)

	for _, files := range packageFiles {
		// Separate test and non-test files
		var testFiles, nonTestFiles []*GoSymbolInfo

		// Extract symbols from all files in the package
		for _, file := range files {
			info, err := ExtractGoSymbols(file)
			if err != nil {
				// Skip files that can't be parsed
				continue
			}

			if strings.HasSuffix(file, "_test.go") {
				testFiles = append(testFiles, info)
			} else {
				nonTestFiles = append(nonTestFiles, info)
			}
		}

		// Build symbol maps separately for test and non-test files
		nonTestSymbolToFiles := make(map[string][]string)
		for _, info := range nonTestFiles {
			for symbol := range info.Defined {
				nonTestSymbolToFiles[symbol] = append(nonTestSymbolToFiles[symbol], info.FilePath)
			}
		}

		allSymbolToFiles := make(map[string][]string)
		for _, info := range append(nonTestFiles, testFiles...) {
			for symbol := range info.Defined {
				allSymbolToFiles[symbol] = append(allSymbolToFiles[symbol], info.FilePath)
			}
		}

		// For non-test files, only allow dependencies on other non-test files
		for _, info := range nonTestFiles {
			deps := make(map[string]bool)
			for symbol := range info.Referenced {
				// Only look in non-test files
				if definingFiles, ok := nonTestSymbolToFiles[symbol]; ok {
					for _, defFile := range definingFiles {
						// Don't add self-dependencies
						if defFile != info.FilePath {
							deps[defFile] = true
						}
					}
				}
			}

			// Convert set to slice
			depSlice := make([]string, 0, len(deps))
			for dep := range deps {
				depSlice = append(depSlice, dep)
			}
			dependencies[info.FilePath] = depSlice
		}

		// For test files, allow dependencies on all files (test and non-test)
		for _, info := range testFiles {
			deps := make(map[string]bool)
			for symbol := range info.Referenced {
				// Look in all files
				if definingFiles, ok := allSymbolToFiles[symbol]; ok {
					for _, defFile := range definingFiles {
						// Don't add self-dependencies
						if defFile != info.FilePath {
							deps[defFile] = true
						}
					}
				}
			}

			// Convert set to slice
			depSlice := make([]string, 0, len(deps))
			for dep := range deps {
				depSlice = append(depSlice, dep)
			}
			dependencies[info.FilePath] = depSlice
		}
	}

	return dependencies, nil
}
