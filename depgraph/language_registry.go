package depgraph

type languageResolverKey string

const (
	languageResolverDart       languageResolverKey = "dart"
	languageResolverGo         languageResolverKey = "go"
	languageResolverJava       languageResolverKey = "java"
	languageResolverKotlin     languageResolverKey = "kotlin"
	languageResolverTypeScript languageResolverKey = "typescript"
)

type languageRegistryEntry struct {
	Name        string
	Extensions  []string
	ResolverKey languageResolverKey
}

// languageRegistry is the single source of truth for supported languages.
// Adding/removing a language should happen here.
var languageRegistry = []languageRegistryEntry{
	{Name: "Dart", Extensions: []string{".dart"}, ResolverKey: languageResolverDart},
	{Name: "Go", Extensions: []string{".go"}, ResolverKey: languageResolverGo},
	{Name: "Java", Extensions: []string{".java"}, ResolverKey: languageResolverJava},
	{Name: "Kotlin", Extensions: []string{".kt", ".kts"}, ResolverKey: languageResolverKotlin},
	{Name: "TypeScript", Extensions: []string{".ts", ".tsx"}, ResolverKey: languageResolverTypeScript},
}

func resolverKeyForExtension(ext string) (languageResolverKey, bool) {
	for _, language := range languageRegistry {
		for _, languageExt := range language.Extensions {
			if languageExt == ext {
				return language.ResolverKey, true
			}
		}
	}

	return "", false
}
