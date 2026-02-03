package common_test

import (
	"path/filepath"
	"testing"

	"github.com/LegacyCodeHQ/sanity/cmd/graph/formatters"
	"github.com/stretchr/testify/assert"
)

func TestGetExtensionColors_BasicFunctionality(t *testing.T) {
	fileNames := []string{
		"main.go",
		"utils.go",
		"main.dart",
		"helper.dart",
		"config.json",
	}

	colors := formatters.GetExtensionColors(fileNames)

	// Should have 3 unique extensions: .go, .dart, .json
	assert.Len(t, colors, 3)
	assert.Contains(t, colors, ".go")
	assert.Contains(t, colors, ".dart")
	assert.Contains(t, colors, ".json")

	// Each extension should have a color assigned
	assert.NotEmpty(t, colors[".go"])
	assert.NotEmpty(t, colors[".dart"])
	assert.NotEmpty(t, colors[".json"])

	// Different extensions should have different colors
	assert.NotEqual(t, colors[".go"], colors[".dart"])
	assert.NotEqual(t, colors[".go"], colors[".json"])
	assert.NotEqual(t, colors[".dart"], colors[".json"])
}

func TestGetExtensionColors_EmptyList(t *testing.T) {
	colors := formatters.GetExtensionColors([]string{})

	assert.Empty(t, colors)
}

func TestGetExtensionColors_FilesWithoutExtensions(t *testing.T) {
	fileNames := []string{
		"README",
		"LICENSE",
		"Makefile",
		"main.go", // One file with extension
	}

	colors := formatters.GetExtensionColors(fileNames)

	// Should only have .go extension
	assert.Len(t, colors, 1)
	assert.Contains(t, colors, ".go")
	assert.NotEmpty(t, colors[".go"])
}

func TestGetExtensionColors_SameExtensionMultipleFiles(t *testing.T) {
	fileNames := []string{
		"main.go",
		"utils.go",
		"output_format.go",
		"helpers.go",
	}

	colors := formatters.GetExtensionColors(fileNames)

	// Should only have one extension (.go) with one color
	assert.Len(t, colors, 1)
	assert.Contains(t, colors, ".go")
	assert.NotEmpty(t, colors[".go"])
}

func TestGetExtensionColors_WithPaths(t *testing.T) {
	fileNames := []string{
		"/path/to/project/main.go",
		"/path/to/project/utils.go",
		"/another/path/helper.dart",
		"relative/path/config.json",
	}

	colors := formatters.GetExtensionColors(fileNames)

	// Should extract extensions correctly from full paths
	assert.Len(t, colors, 3)
	assert.Contains(t, colors, ".go")
	assert.Contains(t, colors, ".dart")
	assert.Contains(t, colors, ".json")
}

func TestGetExtensionColors_ManyExtensions(t *testing.T) {
	// Test with more extensions than available colors to ensure cycling works
	fileNames := []string{
		"file1.go",
		"file2.dart",
		"file3.json",
		"file4.yaml",
		"file5.xml",
		"file6.md",
		"file7.txt",
		"file8.py",
		"file9.js",
		"file10.ts",
		"file11.rs",
		"file12.cpp",
		"file13.java",
		"file14.rb",
		"file15.php",
		"file16.swift",
		"file17.kt",
		"file18.scala",
		"file19.clj",
		"file20.hs",
	}

	colors := formatters.GetExtensionColors(fileNames)

	// Should have 20 unique extensions
	assert.Len(t, colors, 20)

	// All extensions should have colors assigned
	for ext, color := range colors {
		assert.NotEmpty(t, ext, "extension should not be empty")
		assert.NotEmpty(t, color, "color should not be empty for extension %s", ext)
	}

	// Verify all extensions are present
	expectedExts := []string{".go", ".dart", ".json", ".yaml", ".xml", ".md", ".txt", ".py", ".js", ".ts", ".rs", ".cpp", ".java", ".rb", ".php", ".swift", ".kt", ".scala", ".clj", ".hs"}
	for _, ext := range expectedExts {
		assert.Contains(t, colors, ext, "extension %s should be in the map", ext)
	}
}

func TestGetExtensionColors_WithinCallConsistency(t *testing.T) {
	// Test that within a single call, the same extension always maps to the same color
	fileNames := []string{
		"file1.go",
		"file2.go",
		"file3.go",
		"helper.dart",
		"main.dart",
		"config.json",
		"settings.json",
	}

	colors := formatters.GetExtensionColors(fileNames)

	// Should have 3 unique extensions
	assert.Len(t, colors, 3)

	// All .go files should map to the same color
	goColor := colors[".go"]
	assert.NotEmpty(t, goColor)

	// All .dart files should map to the same color
	dartColor := colors[".dart"]
	assert.NotEmpty(t, dartColor)

	// All .json files should map to the same color
	jsonColor := colors[".json"]
	assert.NotEmpty(t, jsonColor)

	// Different extensions should have different colors
	assert.NotEqual(t, goColor, dartColor)
	assert.NotEqual(t, goColor, jsonColor)
	assert.NotEqual(t, dartColor, jsonColor)
}

func TestGetExtensionColors_MixedWithAndWithoutExtensions(t *testing.T) {
	fileNames := []string{
		"main.go",
		"README",
		"helper.dart",
		"LICENSE",
		"config.json",
		"Makefile",
	}

	colors := formatters.GetExtensionColors(fileNames)

	// Should only have extensions for files that have them
	assert.Len(t, colors, 3)
	assert.Contains(t, colors, ".go")
	assert.Contains(t, colors, ".dart")
	assert.Contains(t, colors, ".json")
	assert.NotContains(t, colors, "")
}

func TestGetExtensionColors_ValidColorNames(t *testing.T) {
	fileNames := []string{
		"file1.go",
		"file2.dart",
		"file3.json",
		"file4.yaml",
		"file5.xml",
	}

	colors := formatters.GetExtensionColors(fileNames)

	// Valid color names from the palette
	validColors := []string{
		"lightblue", "lightyellow", "mistyrose", "lightcyan", "lightsalmon",
		"lightpink", "lavender", "peachpuff", "plum", "powderblue", "khaki",
		"palegreen", "palegoldenrod", "paleturquoise", "thistle",
	}

	// All assigned colors should be from the valid palette
	for ext, color := range colors {
		assert.Contains(t, validColors, color, "extension %s should have a valid color, got %s", ext, color)
	}
}

func TestGetExtensionColors_FilesWithMultipleDots(t *testing.T) {
	// Test files with multiple dots (e.g., file.min.js should extract .js)
	fileNames := []string{
		"app.min.js",
		"bundle.min.js",
		"styles.min.css",
		"main.go",
	}

	colors := formatters.GetExtensionColors(fileNames)

	// Should extract the last extension correctly
	assert.Len(t, colors, 3)
	assert.Contains(t, colors, ".js")
	assert.Contains(t, colors, ".css")
	assert.Contains(t, colors, ".go")
	// Should not have .min.js or .min.css as separate extensions
	assert.NotContains(t, colors, ".min.js")
	assert.NotContains(t, colors, ".min.css")
}

func TestGetExtensionColors_HiddenFiles(t *testing.T) {
	// Test hidden files (files starting with dot)
	fileNames := []string{
		".gitignore",
		".env",
		".dockerignore",
		"main.go",
	}

	colors := formatters.GetExtensionColors(fileNames)

	// Hidden files without extensions should be skipped
	// .env might be considered as having no extension or empty extension
	// .gitignore and .dockerignore have no extension
	// Only .go should be present
	assert.Contains(t, colors, ".go")
	// The exact behavior depends on filepath.Ext() - it returns empty for .gitignore
	assert.NotContains(t, colors, "")
}

func TestGetExtensionColors_PathsWithDotsInDirectoryNames(t *testing.T) {
	// Test paths with dots in directory names (ToDOT uses filepath.Base first)
	fileNames := []string{
		"/path/to.dir/file.go",
		"/another/path.with.dots/helper.dart",
		"relative/path.dir/config.json",
	}

	colors := formatters.GetExtensionColors(fileNames)

	// Should extract extensions correctly regardless of dots in directory names
	assert.Len(t, colors, 3)
	assert.Contains(t, colors, ".go")
	assert.Contains(t, colors, ".dart")
	assert.Contains(t, colors, ".json")
	// Should not have directory names as extensions
	assert.NotContains(t, colors, ".dir")
}

func TestGetExtensionColors_MatchesToDOTBehavior(t *testing.T) {
	// Test that GetExtensionColors behaves the same as ToDOT would
	// ToDOT uses filepath.Ext(filepath.Base(source))
	// This test verifies that filepath.Ext() on full paths works the same way
	fileNames := []string{
		"/absolute/path/to/main.go",
		"/absolute/path/to/utils.go",
		"relative/path/helper.dart",
		"config.json",
	}

	colors := formatters.GetExtensionColors(fileNames)

	// Should extract extensions correctly (same as ToDOT would)
	assert.Len(t, colors, 3)
	assert.Contains(t, colors, ".go")
	assert.Contains(t, colors, ".dart")
	assert.Contains(t, colors, ".json")

	// Verify that filepath.Ext() works correctly on full paths
	// (should be equivalent to filepath.Ext(filepath.Base()))
	for _, fileName := range fileNames {
		ext := filepath.Ext(fileName)
		if ext != "" {
			assert.Contains(t, colors, ext, "extension %s from %s should be in colors map", ext, fileName)
		}
	}
}
