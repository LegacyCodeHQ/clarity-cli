package swift

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSwiftImports(t *testing.T) {
	source := `
import Foundation
import MyModule
`
	imports, err := ParseSwiftImports([]byte(source))

	require.NoError(t, err)
	assert.Len(t, imports, 2)
	assert.Equal(t, "Foundation", imports[0].Path)
	assert.Equal(t, "MyModule", imports[1].Path)
}

func TestSwiftImports_ValidFile(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "App.swift")

	content := `
import SwiftUI
`
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	require.NoError(t, err)

	imports, err := SwiftImports(tmpFile)

	require.NoError(t, err)
	assert.Len(t, imports, 1)
	assert.Equal(t, "SwiftUI", imports[0].Path)
}

func TestExtractSwiftTypeIdentifiers_SimpleIdentifier(t *testing.T) {
	source := `
import CodexBarCore

struct FactoryStatusProbeFetchTests {
    func testProbe() {
        let probe = FactoryStatusProbe()
        _ = probe
    }
}
`
	identifiers := ExtractSwiftTypeIdentifiers([]byte(source))
	assert.Contains(t, identifiers, "FactoryStatusProbe")
}

func TestExtractSwiftReferencedSymbols_SkipsMemberAndImplicitCatchNames(t *testing.T) {
	source := []byte(`
enum Severity {
    case error
}

func run(logger: Logger) {
    logger.error("bad")
    let severity: Severity = .error
    do {
        try work()
    } catch {
        print("\(error)")
    }
}
`)

	references := ExtractSwiftReferencedSymbols(source)

	assert.NotContains(t, references, "error")
	assert.NotContains(t, references, "Severity")
	assert.Contains(t, references, "Logger")
	assert.Contains(t, references, "work")
}

func TestParseSwiftTopLevelTypeNames_SkipsExtensions(t *testing.T) {
	source := `
extension SettingsStore {
    var foo: Int { 1 }
}

struct RealType {
    let value: Int
}
`
	types := ParseSwiftTopLevelTypeNames([]byte(source))
	assert.Contains(t, types, "RealType")
	assert.NotContains(t, types, "SettingsStore")
}

func TestParseSwiftTopLevelTypeNames_WithProtocolConformance(t *testing.T) {
	source := `
import Foundation

struct GraphFileNode: Identifiable, Hashable {
    let id: String
}
`
	types := ParseSwiftTopLevelTypeNames([]byte(source))
	assert.Contains(t, types, "GraphFileNode")
}

func TestSwiftParameterReferencesAreScoped(t *testing.T) {
	cases := []struct {
		name   string
		source string
	}{
		{"single name", `func sample(pcm: Data) { _ = pcm.count }`},
		{"external label", `func sample(of pcm: Data) { _ = pcm.count }`},
		{"unlabeled", `func sample(_ pcm: Data) { _ = pcm.count }`},
		{"initializer", `struct Sample { init(pcm: Data) { _ = pcm.count } }`},
		{"subscript", `struct Sample { subscript(pcm: Int) -> Int { pcm } }`},
		{"closure", `func sample() { use { pcm in pcm.count } }`},
		{"typed closure", `func sample() { use { (pcm: Data) in pcm.count } }`},
		{"captured parameter", `func sample(pcm: Data) { use { pcm.count } }`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := []byte(tc.source)
			assert.NotContains(t, ExtractSwiftReferencedSymbols(source), "pcm")
			for _, ref := range ExtractSwiftReferenceLocations(source) {
				assert.NotEqual(t, "pcm", ref.Name)
			}
		})
	}
}

func TestSwiftParameterShadowingPreservesExternalReferences(t *testing.T) {
	source := []byte(`func sample(of pcm: PCM = pcm()) { _ = pcm.count }
func other() { _ = pcm() }
`)
	assert.Contains(t, ExtractSwiftReferencedSymbols(source), "pcm")
	assert.Contains(t, ExtractSwiftReferencedSymbols(source), "PCM", "parameter types remain dependencies")
	var lines []int
	for _, ref := range ExtractSwiftReferenceLocations(source) {
		if ref.Name == "pcm" {
			lines = append(lines, ref.Line)
		}
	}
	assert.Equal(t, []int{1, 2}, lines, "only the default value and the other function reference the global helper")
}

func TestSwiftParameterLabelsAreNotReferences(t *testing.T) {
	source := []byte(`func sample(pcm buffer: Data) { _ = buffer.count }`)
	assert.NotContains(t, ExtractSwiftReferencedSymbols(source), "pcm")
	assert.NotContains(t, ExtractSwiftReferencedSymbols(source), "buffer")
	for _, ref := range ExtractSwiftReferenceLocations(source) {
		assert.NotEqual(t, "pcm", ref.Name)
		assert.NotEqual(t, "buffer", ref.Name)
	}
}

func TestSwiftParameterTypeLikeNameIsNotATypeQualifier(t *testing.T) {
	source := []byte(`func sample(PCM: Buffer) { _ = PCM.count }`)
	assert.NotContains(t, ExtractSwiftReferencedSymbols(source), "PCM")
	assert.Contains(t, ExtractSwiftReferencedSymbols(source), "Buffer")
	assert.Empty(t, ExtractSwiftQualifiedReferences(source))
	for _, ref := range ExtractSwiftReferenceLocations(source) {
		assert.NotEqual(t, "PCM", ref.Name)
		assert.NotEqual(t, "PCM", ref.Qualifier)
	}
}
