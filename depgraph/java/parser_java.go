package java

import (
	"regexp"
	"strings"
)

// JavaImport represents a Java import in source code.
type JavaImport interface {
	Path() string
	IsWildcard() bool
	Package() string
}

// StandardLibraryImport represents a Java/JDK standard library import.
type StandardLibraryImport struct {
	path       string
	isWildcard bool
}

func (s StandardLibraryImport) Path() string {
	return s.path
}

func (s StandardLibraryImport) IsWildcard() bool {
	return s.isWildcard
}

func (s StandardLibraryImport) Package() string {
	return javaImportPackage(s.path)
}

// ExternalImport represents a third-party import.
type ExternalImport struct {
	path       string
	isWildcard bool
}

func (e ExternalImport) Path() string {
	return e.path
}

func (e ExternalImport) IsWildcard() bool {
	return e.isWildcard
}

func (e ExternalImport) Package() string {
	return javaImportPackage(e.path)
}

// InternalImport represents an internal project import.
type InternalImport struct {
	path       string
	isWildcard bool
}

func (i InternalImport) Path() string {
	return i.path
}

func (i InternalImport) IsWildcard() bool {
	return i.isWildcard
}

func (i InternalImport) Package() string {
	return javaImportPackage(i.path)
}

var (
	packagePattern = regexp.MustCompile(`(?m)^\s*package\s+([A-Za-z_][A-Za-z0-9_]*(?:\.[A-Za-z_][A-Za-z0-9_]*)*)\s*;`)
	importPattern  = regexp.MustCompile(`(?m)^\s*import\s+(?:static\s+)?([A-Za-z_][A-Za-z0-9_]*(?:\.[A-Za-z_][A-Za-z0-9_]*)*(?:\.\*)?)\s*;`)
	typePattern    = regexp.MustCompile(`\b(?:class|interface|enum|record)\s+([A-Za-z_][A-Za-z0-9_]*)\b|@interface\s+([A-Za-z_][A-Za-z0-9_]*)\b`)
)

// ParsePackageDeclaration extracts the Java package from source code.
func ParsePackageDeclaration(sourceCode []byte) string {
	match := packagePattern.FindSubmatch(sourceCode)
	if len(match) < 2 {
		return ""
	}
	return strings.TrimSpace(string(match[1]))
}

// ParseTopLevelTypeNames extracts declared type names from Java source code.
func ParseTopLevelTypeNames(sourceCode []byte) []string {
	matches := typePattern.FindAllSubmatch(sourceCode, -1)
	if len(matches) == 0 {
		return []string{}
	}

	seen := make(map[string]bool)
	result := make([]string, 0, len(matches))
	for _, m := range matches {
		name := ""
		if len(m) > 1 && len(m[1]) > 0 {
			name = string(m[1])
		}
		if name == "" && len(m) > 2 && len(m[2]) > 0 {
			name = string(m[2])
		}
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		result = append(result, name)
	}

	return result
}

// ParseJavaImports parses Java source code and classifies imports.
func ParseJavaImports(sourceCode []byte, projectPackages map[string]bool) []JavaImport {
	matches := importPattern.FindAllSubmatch(sourceCode, -1)
	if len(matches) == 0 {
		return []JavaImport{}
	}

	imports := make([]JavaImport, 0, len(matches))
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		path := strings.TrimSpace(string(m[1]))
		if path == "" {
			continue
		}
		imports = append(imports, classifyJavaImport(path, projectPackages))
	}

	return imports
}

func classifyJavaImport(importPath string, projectPackages map[string]bool) JavaImport {
	isWildcard := strings.HasSuffix(importPath, ".*")
	if isStandardLibraryImport(importPath) {
		return StandardLibraryImport{path: importPath, isWildcard: isWildcard}
	}

	if isInternalJavaImport(importPath, projectPackages) {
		return InternalImport{path: importPath, isWildcard: isWildcard}
	}

	return ExternalImport{path: importPath, isWildcard: isWildcard}
}

func isStandardLibraryImport(path string) bool {
	prefixes := []string{
		"java.",
		"javax.",
		"jdk.",
		"sun.",
		"com.sun.",
		"org.w3c.",
		"org.xml.sax.",
	}

	for _, prefix := range prefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func isInternalJavaImport(importPath string, projectPackages map[string]bool) bool {
	pkg := javaImportPackage(importPath)
	if projectPackages[pkg] || projectPackages[importPath] {
		return true
	}

	for projectPkg := range projectPackages {
		if strings.HasPrefix(pkg, projectPkg+".") || pkg == projectPkg {
			return true
		}
		if strings.HasPrefix(importPath, projectPkg+".") {
			return true
		}
	}

	return false
}

func javaImportPackage(path string) string {
	trimmed := strings.TrimSuffix(path, ".*")
	parts := strings.Split(trimmed, ".")
	if len(parts) <= 1 {
		return trimmed
	}

	last := parts[len(parts)-1]
	if len(last) > 0 {
		first := last[0]
		if first >= 'A' && first <= 'Z' {
			return strings.Join(parts[:len(parts)-1], ".")
		}
	}

	return trimmed
}

func simpleTypeName(path string) string {
	trimmed := strings.TrimSuffix(path, ".*")
	parts := strings.Split(trimmed, ".")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}
