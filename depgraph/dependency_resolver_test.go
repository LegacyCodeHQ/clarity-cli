package depgraph

import (
	"testing"

	"github.com/LegacyCodeHQ/clarity/depgraph/registry"
)

func TestDefaultResolverSupportsAllRegisteredLanguageExtensions(t *testing.T) {
	resolver := NewDefaultDependencyResolver(&dependencyGraphContext{}, nil)

	for _, ext := range registry.SupportedLanguageExtensions() {
		if !resolver.SupportsFileExtension(ext) {
			t.Fatalf("resolver does not support registered extension %q", ext)
		}
	}
}
