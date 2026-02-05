package depgraph

import "testing"

func TestSupportedLanguages(t *testing.T) {
	languages := SupportedLanguages()
	if len(languages) == 0 {
		t.Fatalf("SupportedLanguages() returned no languages")
	}

	foundTypeScript := false
	for _, language := range languages {
		if language.Name != "TypeScript" {
			continue
		}
		foundTypeScript = true
		if len(language.Extensions) != 2 {
			t.Fatalf("TypeScript extension count = %d, want 2", len(language.Extensions))
		}
	}

	if !foundTypeScript {
		t.Fatalf("SupportedLanguages() missing TypeScript")
	}
}

func TestIsSupportedLanguageExtension(t *testing.T) {
	if !IsSupportedLanguageExtension(".go") {
		t.Fatalf("IsSupportedLanguageExtension(.go) = false, want true")
	}
	if !IsSupportedLanguageExtension(".kts") {
		t.Fatalf("IsSupportedLanguageExtension(.kts) = false, want true")
	}
	if IsSupportedLanguageExtension(".md") {
		t.Fatalf("IsSupportedLanguageExtension(.md) = true, want false")
	}
}
