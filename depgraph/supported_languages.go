package depgraph

import "github.com/LegacyCodeHQ/clarity/depgraph/registry"

// LanguageSupport describes one supported programming language and
// the file extensions that map to it.
type LanguageSupport = registry.LanguageSupport

// SupportedLanguages returns a copy of all supported languages and their extensions.
func SupportedLanguages() []LanguageSupport {
	return registry.SupportedLanguages()
}

// IsSupportedLanguageExtension reports whether Clarity can analyze files with the extension.
func IsSupportedLanguageExtension(ext string) bool {
	return registry.IsSupportedLanguageExtension(ext)
}

// SupportedLanguageExtensions returns all supported language extensions in sorted order.
func SupportedLanguageExtensions() []string {
	return registry.SupportedLanguageExtensions()
}
