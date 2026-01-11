package parsers

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildDependencyGraph(t *testing.T) {
	// Create temporary directory with test files
	tmpDir := t.TempDir()

	// Create main.dart
	mainContent := `
		import 'dart:io';
		import 'package:flutter/material.dart';
		import 'models/user.dart';
		import 'services/api.dart';

		void main() {}
	`
	mainPath := filepath.Join(tmpDir, "main.dart")
	err := os.WriteFile(mainPath, []byte(mainContent), 0644)
	require.NoError(t, err)

	// Create models directory and user.dart
	modelsDir := filepath.Join(tmpDir, "models")
	err = os.Mkdir(modelsDir, 0755)
	require.NoError(t, err)

	userContent := `
		import '../utils/validator.dart';

		class User {
			String name;
		}
	`
	userPath := filepath.Join(modelsDir, "user.dart")
	err = os.WriteFile(userPath, []byte(userContent), 0644)
	require.NoError(t, err)

	// Create services directory and api.dart
	servicesDir := filepath.Join(tmpDir, "services")
	err = os.Mkdir(servicesDir, 0755)
	require.NoError(t, err)

	apiContent := `
		import 'package:http/http.dart';
		import '../models/user.dart';

		class Api {}
	`
	apiPath := filepath.Join(servicesDir, "api.dart")
	err = os.WriteFile(apiPath, []byte(apiContent), 0644)
	require.NoError(t, err)

	// Create utils directory and validator.dart
	utilsDir := filepath.Join(tmpDir, "utils")
	err = os.Mkdir(utilsDir, 0755)
	require.NoError(t, err)

	validatorContent := `
		class Validator {}
	`
	validatorPath := filepath.Join(utilsDir, "validator.dart")
	err = os.WriteFile(validatorPath, []byte(validatorContent), 0644)
	require.NoError(t, err)

	// Build dependency graph
	files := []string{mainPath, userPath, apiPath, validatorPath}
	graph, err := BuildDependencyGraph(files, "", "")

	require.NoError(t, err)
	assert.Len(t, graph, 4)

	// Check main.dart dependencies (should have 2 project imports)
	mainDeps := graph[mainPath]
	assert.Len(t, mainDeps, 2)
	assert.Contains(t, mainDeps, userPath)
	assert.Contains(t, mainDeps, apiPath)

	// Check user.dart dependencies (should have 1 project import)
	userDeps := graph[userPath]
	assert.Len(t, userDeps, 1)
	assert.Contains(t, userDeps, validatorPath)

	// Check api.dart dependencies (should have 1 project import)
	apiDeps := graph[apiPath]
	assert.Len(t, apiDeps, 1)
	assert.Contains(t, apiDeps, userPath)

	// Check validator.dart dependencies (should have none)
	validatorDeps := graph[validatorPath]
	assert.Empty(t, validatorDeps)
}

func TestBuildDependencyGraph_EmptyFileList(t *testing.T) {
	graph, err := BuildDependencyGraph([]string{}, "", "")

	require.NoError(t, err)
	assert.Empty(t, graph)
}

func TestBuildDependencyGraph_NonexistentFile(t *testing.T) {
	_, err := BuildDependencyGraph([]string{"/nonexistent/file.dart"}, "", "")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse imports")
}

func TestBuildDependencyGraph_FiltersNonSuppliedFiles(t *testing.T) {
	// Create temporary directory with test files
	tmpDir := t.TempDir()

	// Create main.dart that imports helper.dart and utils.dart
	mainContent := `
		import 'helper.dart';
		import 'utils.dart';

		void main() {}
	`
	mainPath := filepath.Join(tmpDir, "main.dart")
	err := os.WriteFile(mainPath, []byte(mainContent), 0644)
	require.NoError(t, err)

	// Create helper.dart (we'll include this in the supplied files)
	helperContent := `
		class Helper {}
	`
	helperPath := filepath.Join(tmpDir, "helper.dart")
	err = os.WriteFile(helperPath, []byte(helperContent), 0644)
	require.NoError(t, err)

	// Create utils.dart (we'll NOT include this in the supplied files)
	utilsContent := `
		class Utils {}
	`
	utilsPath := filepath.Join(tmpDir, "utils.dart")
	err = os.WriteFile(utilsPath, []byte(utilsContent), 0644)
	require.NoError(t, err)

	// Build dependency graph with only main.dart and helper.dart
	// (utils.dart is NOT supplied, so it should be filtered out)
	files := []string{mainPath, helperPath}
	graph, err := BuildDependencyGraph(files, "", "")

	require.NoError(t, err)
	assert.Len(t, graph, 2)

	// Check main.dart dependencies (should only contain helper.dart, NOT utils.dart)
	mainDeps := graph[mainPath]
	assert.Len(t, mainDeps, 1, "main.dart should only have 1 dependency (helper.dart)")
	assert.Contains(t, mainDeps, helperPath)
	assert.NotContains(t, mainDeps, utilsPath, "utils.dart should be filtered out since it wasn't supplied")

	// Check helper.dart dependencies (should have none)
	helperDeps := graph[helperPath]
	assert.Empty(t, helperDeps)
}

func TestDependencyGraph_ToJSON(t *testing.T) {
	graph := DependencyGraph{
		"/project/main.dart":  {"/project/utils.dart", "/project/models/user.dart"},
		"/project/utils.dart": {},
	}

	jsonData, err := graph.ToJSON()

	require.NoError(t, err)
	assert.Contains(t, string(jsonData), "/project/main.dart")
	assert.Contains(t, string(jsonData), "/project/utils.dart")
	assert.Contains(t, string(jsonData), "/project/models/user.dart")
}

func TestDependencyGraph_ToDOT(t *testing.T) {
	graph := DependencyGraph{
		"/project/main.dart":  {"/project/utils.dart"},
		"/project/utils.dart": {},
	}

	dot := graph.ToDOT()

	assert.Contains(t, dot, "digraph dependencies")
	assert.Contains(t, dot, "main.dart")
	assert.Contains(t, dot, "utils.dart")
	assert.Contains(t, dot, "->")
}

func TestBuildDependencyGraph_IncludesNonDartFiles(t *testing.T) {
	// Create temporary directory with test files
	tmpDir := t.TempDir()

	// Create a .dart file
	dartContent := `
		import 'dart:io';
		void main() {}
	`
	dartPath := filepath.Join(tmpDir, "main.dart")
	err := os.WriteFile(dartPath, []byte(dartContent), 0644)
	require.NoError(t, err)

	// Create a non-.dart file (Go file)
	goPath := filepath.Join(tmpDir, "main.go")
	err = os.WriteFile(goPath, []byte("package main"), 0644)
	require.NoError(t, err)

	// Create another non-.dart file (README)
	readmePath := filepath.Join(tmpDir, "README.md")
	err = os.WriteFile(readmePath, []byte("# Test"), 0644)
	require.NoError(t, err)

	// Build dependency graph with all files
	files := []string{dartPath, goPath, readmePath}
	graph, err := BuildDependencyGraph(files, "", "")

	require.NoError(t, err)
	assert.Len(t, graph, 3, "graph should include all files")

	// Verify .dart file is in the graph (may have dependencies parsed)
	assert.Contains(t, graph, dartPath)

	// Verify non-.dart files are in the graph with empty dependencies
	assert.Contains(t, graph, goPath)
	assert.Empty(t, graph[goPath], "non-dart file should have no dependencies")

	assert.Contains(t, graph, readmePath)
	assert.Empty(t, graph[readmePath], "non-dart file should have no dependencies")
}

func TestBuildDependencyGraph_GoFiles(t *testing.T) {
	// Create temporary directory with test Go files
	tmpDir := t.TempDir()

	// Create go.mod
	goModContent := `module testproject

go 1.25
`
	goModPath := filepath.Join(tmpDir, "go.mod")
	err := os.WriteFile(goModPath, []byte(goModContent), 0644)
	require.NoError(t, err)

	// Create main.go
	mainContent := `package main

import (
	"fmt"
	"testproject/models"
	"testproject/services"
)

func main() {
	fmt.Println("Hello")
}
`
	mainPath := filepath.Join(tmpDir, "main.go")
	err = os.WriteFile(mainPath, []byte(mainContent), 0644)
	require.NoError(t, err)

	// Create models directory and user.go
	modelsDir := filepath.Join(tmpDir, "models")
	err = os.Mkdir(modelsDir, 0755)
	require.NoError(t, err)

	userContent := `package models

import "testproject/utils"

type User struct {
	Name string
}
`
	userPath := filepath.Join(modelsDir, "user.go")
	err = os.WriteFile(userPath, []byte(userContent), 0644)
	require.NoError(t, err)

	// Create services directory and api.go
	servicesDir := filepath.Join(tmpDir, "services")
	err = os.Mkdir(servicesDir, 0755)
	require.NoError(t, err)

	apiContent := `package services

import (
	"net/http"
	"testproject/models"
)

type Api struct {}
`
	apiPath := filepath.Join(servicesDir, "api.go")
	err = os.WriteFile(apiPath, []byte(apiContent), 0644)
	require.NoError(t, err)

	// Create utils directory and validator.go
	utilsDir := filepath.Join(tmpDir, "utils")
	err = os.Mkdir(utilsDir, 0755)
	require.NoError(t, err)

	validatorContent := `package utils

type Validator struct {}
`
	validatorPath := filepath.Join(utilsDir, "validator.go")
	err = os.WriteFile(validatorPath, []byte(validatorContent), 0644)
	require.NoError(t, err)

	// Build dependency graph
	// Note: Go imports refer to packages (directories), but the graph maps
	// file to file dependencies (all files in the imported package)
	files := []string{mainPath, userPath, apiPath, validatorPath}
	graph, err := BuildDependencyGraph(files, "", "")

	require.NoError(t, err)
	assert.Len(t, graph, 4)

	// Check main.go dependencies (should reference user.go and api.go files)
	mainDeps := graph[mainPath]
	assert.Len(t, mainDeps, 2)
	assert.Contains(t, mainDeps, userPath)
	assert.Contains(t, mainDeps, apiPath)

	// Check user.go dependencies (should reference validator.go file)
	userDeps := graph[userPath]
	assert.Len(t, userDeps, 1)
	assert.Contains(t, userDeps, validatorPath)

	// Check api.go dependencies (should reference user.go file)
	apiDeps := graph[apiPath]
	assert.Len(t, apiDeps, 1)
	assert.Contains(t, apiDeps, userPath)

	// Check validator.go dependencies (should have none)
	validatorDeps := graph[validatorPath]
	assert.Empty(t, validatorDeps)
}

func TestBuildDependencyGraph_MixedDartAndGo(t *testing.T) {
	// Create temporary directory with mixed files
	tmpDir := t.TempDir()

	// Create go.mod for Go support
	goModContent := `module mixedproject

go 1.25
`
	goModPath := filepath.Join(tmpDir, "go.mod")
	err := os.WriteFile(goModPath, []byte(goModContent), 0644)
	require.NoError(t, err)

	// Create a Dart file
	dartContent := `
		import 'dart:io';
		import 'helper.dart';

		void main() {}
	`
	dartPath := filepath.Join(tmpDir, "main.dart")
	err = os.WriteFile(dartPath, []byte(dartContent), 0644)
	require.NoError(t, err)

	helperContent := `
		class Helper {}
	`
	helperPath := filepath.Join(tmpDir, "helper.dart")
	err = os.WriteFile(helperPath, []byte(helperContent), 0644)
	require.NoError(t, err)

	// Create a Go file
	goContent := `package main

import (
	"fmt"
	"mixedproject/utils"
)

func main() {}
`
	goPath := filepath.Join(tmpDir, "main.go")
	err = os.WriteFile(goPath, []byte(goContent), 0644)
	require.NoError(t, err)

	// Create utils directory
	utilsDir := filepath.Join(tmpDir, "utils")
	err = os.Mkdir(utilsDir, 0755)
	require.NoError(t, err)

	utilsContent := `package utils

func Helper() {}
`
	utilsPath := filepath.Join(utilsDir, "helper.go")
	err = os.WriteFile(utilsPath, []byte(utilsContent), 0644)
	require.NoError(t, err)

	// Build dependency graph with both Dart and Go files
	files := []string{dartPath, helperPath, goPath, utilsPath}
	graph, err := BuildDependencyGraph(files, "", "")

	require.NoError(t, err)
	assert.Len(t, graph, 4)

	// Check Dart file dependencies
	dartDeps := graph[dartPath]
	assert.Len(t, dartDeps, 1)
	assert.Contains(t, dartDeps, helperPath)

	// Check Go file dependencies (should reference helper.go file in utils package)
	goDeps := graph[goPath]
	assert.Len(t, goDeps, 1)
	assert.Contains(t, goDeps, utilsPath)
}

func TestBuildDependencyGraph_GoSymbolLevel(t *testing.T) {
	// Create temporary directory with Go files in same package
	tmpDir := t.TempDir()

	// Create go.mod
	goModContent := `module symboltest

go 1.25
`
	goModPath := filepath.Join(tmpDir, "go.mod")
	err := os.WriteFile(goModPath, []byte(goModContent), 0644)
	require.NoError(t, err)

	// Create types.go with type definitions
	typesContent := `package main

type User struct {
	Name string
}

type Product struct {
	Title string
}
`
	typesPath := filepath.Join(tmpDir, "types.go")
	err = os.WriteFile(typesPath, []byte(typesContent), 0644)
	require.NoError(t, err)

	// Create helpers.go with helper functions
	helpersContent := `package main

func FormatUser(u User) string {
	return u.Name
}
`
	helpersPath := filepath.Join(tmpDir, "helpers.go")
	err = os.WriteFile(helpersPath, []byte(helpersContent), 0644)
	require.NoError(t, err)

	// Create main.go that uses both User and FormatUser
	mainContent := `package main

import "fmt"

func main() {
	u := User{Name: "Alice"}
	fmt.Println(FormatUser(u))

	p := Product{Title: "Book"}
	fmt.Println(p.Title)
}
`
	mainPath := filepath.Join(tmpDir, "main.go")
	err = os.WriteFile(mainPath, []byte(mainContent), 0644)
	require.NoError(t, err)

	// Build dependency graph
	files := []string{typesPath, helpersPath, mainPath}
	graph, err := BuildDependencyGraph(files, "", "")

	require.NoError(t, err)
	assert.Len(t, graph, 3)

	// Check main.go dependencies (should depend on both types.go and helpers.go)
	// because it uses User and Product from types.go, and FormatUser from helpers.go
	mainDeps := graph[mainPath]
	assert.Len(t, mainDeps, 2, "main.go should depend on both types.go and helpers.go")
	assert.Contains(t, mainDeps, typesPath, "main.go uses User and Product")
	assert.Contains(t, mainDeps, helpersPath, "main.go uses FormatUser")

	// Check helpers.go dependencies (should depend on types.go)
	// because it uses User type
	helpersDeps := graph[helpersPath]
	assert.Len(t, helpersDeps, 1, "helpers.go should depend on types.go")
	assert.Contains(t, helpersDeps, typesPath, "helpers.go uses User type")

	// Check types.go dependencies (should have none)
	typesDeps := graph[typesPath]
	assert.Empty(t, typesDeps, "types.go has no dependencies")
}
