package kotlin

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseKotlinImports_BasicImports(t *testing.T) {
	source := []byte(`
package com.example.app

import kotlin.collections.List
import java.util.Date
import com.google.gson.Gson
import com.example.models.User

class MainActivity {
    fun main() {}
}
`)

	imports, err := ParseKotlinImports(source)

	require.NoError(t, err)
	assert.Len(t, imports, 4)

	// Verify import paths
	paths := make([]string, len(imports))
	for i, imp := range imports {
		paths[i] = imp.Path()
	}
	assert.Contains(t, paths, "kotlin.collections.List")
	assert.Contains(t, paths, "java.util.Date")
	assert.Contains(t, paths, "com.google.gson.Gson")
	assert.Contains(t, paths, "com.example.models.User")
}

func TestParseKotlinImports_WildcardImports(t *testing.T) {
	source := []byte(`
package com.example.app

import kotlin.collections.*
import com.example.models.*

class MainActivity
`)

	imports, err := ParseKotlinImports(source)

	require.NoError(t, err)
	assert.Len(t, imports, 2)

	// Check wildcard flag
	for _, imp := range imports {
		assert.True(t, imp.IsWildcard(), "Import %s should be wildcard", imp.Path())
	}

	// Check paths (should have .* stripped)
	paths := make([]string, len(imports))
	for i, imp := range imports {
		paths[i] = imp.Path()
	}
	assert.Contains(t, paths, "kotlin.collections")
	assert.Contains(t, paths, "com.example.models")
}

func TestParseKotlinImports_AliasedImports(t *testing.T) {
	source := []byte(`
package com.example.app

import com.example.Foo as Bar
import kotlin.collections.List as KList

class MainActivity
`)

	imports, err := ParseKotlinImports(source)

	require.NoError(t, err)
	assert.Len(t, imports, 2)

	// Verify paths (alias should be ignored)
	paths := make([]string, len(imports))
	for i, imp := range imports {
		paths[i] = imp.Path()
	}
	assert.Contains(t, paths, "com.example.Foo")
	assert.Contains(t, paths, "kotlin.collections.List")
}

func TestClassifyKotlinImport_StandardLibrary(t *testing.T) {
	projectPackages := map[string]bool{
		"com.example.app": true,
	}

	tests := []struct {
		name       string
		importPath string
		isWildcard bool
	}{
		{"kotlin stdlib", "kotlin.collections.List", false},
		{"kotlin stdlib wildcard", "kotlin.collections", true},
		{"kotlinx", "kotlinx.coroutines.Dispatchers", false},
		{"java stdlib", "java.util.Date", false},
		{"javax", "javax.inject.Inject", false},
		{"android", "android.os.Bundle", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imp := classifyKotlinImport(tt.importPath, tt.isWildcard, projectPackages)
			assert.IsType(t, StandardLibraryImport{}, imp, "Expected StandardLibraryImport for %s", tt.importPath)
			assert.Equal(t, tt.importPath, imp.Path())
			assert.Equal(t, tt.isWildcard, imp.IsWildcard())
		})
	}
}

func TestClassifyKotlinImport_External(t *testing.T) {
	projectPackages := map[string]bool{
		"com.example.app": true,
	}

	tests := []struct {
		name       string
		importPath string
		isWildcard bool
	}{
		{"google library", "com.google.gson.Gson", false},
		{"junit", "org.junit.Test", false},
		{"retrofit", "retrofit2.http.GET", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imp := classifyKotlinImport(tt.importPath, tt.isWildcard, projectPackages)
			assert.IsType(t, ExternalImport{}, imp, "Expected ExternalImport for %s", tt.importPath)
			assert.Equal(t, tt.importPath, imp.Path())
			assert.Equal(t, tt.isWildcard, imp.IsWildcard())
		})
	}
}

func TestClassifyKotlinImport_Internal(t *testing.T) {
	projectPackages := map[string]bool{
		"com.example.app":        true,
		"com.example.app.models": true,
	}

	tests := []struct {
		name       string
		importPath string
		isWildcard bool
	}{
		{"exact package match", "com.example.app", true},
		{"sub-package", "com.example.app.models.User", false},
		{"another sub-package", "com.example.app.viewmodels", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imp := classifyKotlinImport(tt.importPath, tt.isWildcard, projectPackages)
			assert.IsType(t, InternalImport{}, imp, "Expected InternalImport for %s", tt.importPath)
			assert.Equal(t, tt.importPath, imp.Path())
			assert.Equal(t, tt.isWildcard, imp.IsWildcard())
		})
	}
}

func TestExtractPackageDeclaration(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		expected string
	}{
		{
			name: "simple package",
			source: `package com.example.app

class MainActivity`,
			expected: "com.example.app",
		},
		{
			name: "package with imports",
			source: `package com.example.app.models

import kotlin.collections.List

class User`,
			expected: "com.example.app.models",
		},
		{
			name: "no package",
			source: `import kotlin.collections.List

class MainActivity`,
			expected: "",
		},
		{
			name:     "empty file",
			source:   "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkg := ExtractPackageDeclaration([]byte(tt.source))
			assert.Equal(t, tt.expected, pkg)
		})
	}
}

func TestExtractPackageFromPath(t *testing.T) {
	tests := []struct {
		name       string
		importPath string
		expected   string
	}{
		{
			name:       "class import",
			importPath: "com.example.app.MainActivity",
			expected:   "com.example.app",
		},
		{
			name:       "package only",
			importPath: "com.example.app",
			expected:   "com.example.app",
		},
		{
			name:       "nested class",
			importPath: "com.example.app.models.User",
			expected:   "com.example.app.models",
		},
		{
			name:       "single level",
			importPath: "kotlin",
			expected:   "kotlin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractPackageFromPath(tt.importPath)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestKotlinImportPackageMethod(t *testing.T) {
	tests := []struct {
		name       string
		importPath string
		isWildcard bool
		expected   string
	}{
		{
			name:       "wildcard import",
			importPath: "com.example.models",
			isWildcard: true,
			expected:   "com.example.models",
		},
		{
			name:       "class import",
			importPath: "com.example.models.User",
			isWildcard: false,
			expected:   "com.example.models",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imp := InternalImport{path: tt.importPath, isWildcard: tt.isWildcard}
			assert.Equal(t, tt.expected, imp.Package())
		})
	}
}

func TestParseKotlinImports_EmptyFile(t *testing.T) {
	source := []byte("")

	imports, err := ParseKotlinImports(source)

	require.NoError(t, err)
	assert.Empty(t, imports)
}

func TestParseKotlinImports_NoImports(t *testing.T) {
	source := []byte(`
package com.example.app

class MainActivity {
    fun main() {
        println("Hello")
    }
}
`)

	imports, err := ParseKotlinImports(source)

	require.NoError(t, err)
	assert.Empty(t, imports)
}

func TestClassifyWithProjectPackages(t *testing.T) {
	// Initial classification without project knowledge
	initialImports := []KotlinImport{
		ExternalImport{path: "com.example.app.User", isWildcard: false},
		ExternalImport{path: "com.example.app.models", isWildcard: true},
		StandardLibraryImport{path: "kotlin.collections.List", isWildcard: false},
	}

	// Project packages
	projectPackages := map[string]bool{
		"com.example.app":        true,
		"com.example.app.models": true,
	}

	// Reclassify
	reclassified := ClassifyWithProjectPackages(initialImports, projectPackages)

	require.Len(t, reclassified, 3)

	// First should be internal now
	assert.IsType(t, InternalImport{}, reclassified[0])
	assert.Equal(t, "com.example.app.User", reclassified[0].Path())

	// Second should be internal now
	assert.IsType(t, InternalImport{}, reclassified[1])
	assert.Equal(t, "com.example.app.models", reclassified[1].Path())

	// Third should still be standard library
	assert.IsType(t, StandardLibraryImport{}, reclassified[2])
	assert.Equal(t, "kotlin.collections.List", reclassified[2].Path())
}

func TestExtractTopLevelTypeNames(t *testing.T) {
	source := []byte(`
package com.example

data class ActivateLicenseRequest(val token: String)

interface LicensingClient

class NestedContainer {
  class InnerClass
}

typealias Token = String
`)

	decls := ExtractTopLevelTypeNames(source)
	assert.Contains(t, decls, "ActivateLicenseRequest")
	assert.Contains(t, decls, "LicensingClient")
	assert.Contains(t, decls, "Token")
	assert.NotContains(t, decls, "InnerClass")
}

func TestExtractTypeIdentifiers(t *testing.T) {
	source := []byte(`
package com.example

fun demo(request: ActivateLicenseRequest): ActivateLicenseResponse {
  val machine: Machine = Machine()
  return ActivateLicenseResponse()
}
`)

	identifiers := ExtractTypeIdentifiers(source)
	assert.Contains(t, identifiers, "ActivateLicenseRequest")
	assert.Contains(t, identifiers, "ActivateLicenseResponse")
	assert.Contains(t, identifiers, "Machine")
}
