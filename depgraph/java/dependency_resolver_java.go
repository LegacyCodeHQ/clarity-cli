package java

import (
	"fmt"
	"path/filepath"

	"github.com/LegacyCodeHQ/sanity/vcs"
)

// BuildJavaIndices builds package and type indices for supplied Java files.
func BuildJavaIndices(
	javaFiles []string,
	contentReader vcs.ContentReader,
) (map[string][]string, map[string]map[string][]string) {
	if len(javaFiles) == 0 {
		return nil, nil
	}

	packageToFiles := make(map[string][]string)
	packageToTypes := make(map[string]map[string][]string)

	for _, filePath := range javaFiles {
		absPath, err := filepath.Abs(filePath)
		if err != nil {
			continue
		}

		content, err := contentReader(absPath)
		if err != nil {
			continue
		}

		pkg := ParsePackageDeclaration(content)
		if pkg == "" {
			continue
		}

		packageToFiles[pkg] = append(packageToFiles[pkg], absPath)

		declaredTypes := ParseTopLevelTypeNames(content)
		if len(declaredTypes) == 0 {
			continue
		}

		typeMap, ok := packageToTypes[pkg]
		if !ok {
			typeMap = make(map[string][]string)
			packageToTypes[pkg] = typeMap
		}
		for _, typeName := range declaredTypes {
			if typeName == "" {
				continue
			}
			typeMap[typeName] = append(typeMap[typeName], absPath)
		}
	}

	return packageToFiles, packageToTypes
}

// ResolveJavaProjectImports resolves Java project imports for a single file.
func ResolveJavaProjectImports(
	absPath string,
	_ string,
	javaPackageIndex map[string][]string,
	javaPackageTypes map[string]map[string][]string,
	suppliedFiles map[string]bool,
	contentReader vcs.ContentReader,
) ([]string, error) {
	content, err := contentReader(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", absPath, err)
	}

	projectPackages := make(map[string]bool, len(javaPackageIndex))
	for pkg := range javaPackageIndex {
		projectPackages[pkg] = true
	}

	imports := ParseJavaImports(content, projectPackages)
	projectImports := make([]string, 0, len(imports))
	for _, imp := range imports {
		internalImp, ok := imp.(InternalImport)
		if !ok {
			continue
		}
		projectImports = append(projectImports, resolveJavaImportPath(absPath, internalImp, javaPackageIndex, javaPackageTypes, suppliedFiles)...)
	}

	return projectImports, nil
}

func resolveJavaImportPath(
	sourceFile string,
	imp InternalImport,
	packageIndex map[string][]string,
	packageTypeIndex map[string]map[string][]string,
	suppliedFiles map[string]bool,
) []string {
	pkg := imp.Package()
	resolved := []string{}
	seen := make(map[string]bool)

	addFile := func(path string) {
		if path == sourceFile || !suppliedFiles[path] || seen[path] {
			return
		}
		seen[path] = true
		resolved = append(resolved, path)
	}

	if imp.IsWildcard() {
		for _, file := range packageIndex[pkg] {
			addFile(file)
		}
		return resolved
	}

	typeName := simpleTypeName(imp.Path())
	if typeName != "" {
		if typeMap, ok := packageTypeIndex[pkg]; ok {
			for _, file := range typeMap[typeName] {
				addFile(file)
			}
		}
	}

	if len(resolved) == 0 {
		for _, file := range packageIndex[pkg] {
			addFile(file)
		}
	}

	return resolved
}
