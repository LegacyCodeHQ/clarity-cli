package langsupport

import "github.com/LegacyCodeHQ/sanity/vcs"
import graphlib "github.com/dominikbraun/graph"

// Graph is the minimal graph contract language resolvers need during finalization.
type Graph interface {
	Vertex(hash string) (string, error)
	AddEdge(sourceHash, targetHash string, options ...func(*graphlib.EdgeProperties)) error
}

// Resolver resolves project imports for one language and can finalize graph-wide state.
type Resolver interface {
	ResolveProjectImports(absPath, filePath, ext string) ([]string, error)
	FinalizeGraph(graph Graph) error
}

// Context contains precomputed project data shared across language resolvers.
type Context struct {
	SuppliedFiles map[string]bool
	DirToFiles    map[string][]string
	JavaFiles     []string
	KotlinFiles   []string
	GoFiles       []string
}

// Module describes pluggable language support.
type Module interface {
	Name() string
	Extensions() []string
	NewResolver(ctx *Context, contentReader vcs.ContentReader) Resolver
	IsTestFile(filePath string, contentReader vcs.ContentReader) bool
}
