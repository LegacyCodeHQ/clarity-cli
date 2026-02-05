package parsers

import (
	"github.com/LegacyCodeHQ/sanity/parsers/dart"
	_go "github.com/LegacyCodeHQ/sanity/parsers/go"
	"github.com/LegacyCodeHQ/sanity/parsers/kotlin"
	"github.com/LegacyCodeHQ/sanity/parsers/typescript"
	"github.com/LegacyCodeHQ/sanity/vcs"
)

// DependencyResolver resolves project imports per file and can finalize graph-wide dependencies.
type DependencyResolver interface {
	ResolveProjectImports(absPath, filePath, ext string) ([]string, error)
	FinalizeGraph(graph DependencyGraph) error
}

type defaultDependencyResolver struct {
	ctx                  *dependencyGraphContext
	contentReader        vcs.ContentReader
	goPackageExportIndex map[string]_go.GoPackageExportIndex
	kotlinPackageIndex   map[string][]string
	kotlinPackageTypes   map[string]map[string][]string
	kotlinFilePackages   map[string]string
}

// NewDefaultDependencyResolver creates the built-in language-aware dependency resolver.
func NewDefaultDependencyResolver(ctx *dependencyGraphContext, contentReader vcs.ContentReader) DependencyResolver {
	goPackageExportIndex := _go.BuildGoPackageExportIndices(ctx.dirToFiles, contentReader)
	kotlinPackageIndex, kotlinPackageTypes, kotlinFilePackages := kotlin.BuildKotlinIndices(ctx.kotlinFiles, contentReader)

	return &defaultDependencyResolver{
		ctx:                  ctx,
		contentReader:        contentReader,
		goPackageExportIndex: goPackageExportIndex,
		kotlinPackageIndex:   kotlinPackageIndex,
		kotlinPackageTypes:   kotlinPackageTypes,
		kotlinFilePackages:   kotlinFilePackages,
	}
}

func (b *defaultDependencyResolver) ResolveProjectImports(absPath, filePath, ext string) ([]string, error) {
	switch ext {
	case ".dart":
		return dart.ResolveDartProjectImports(absPath, filePath, ext, b.ctx.suppliedFiles, b.contentReader)
	case ".go":
		return _go.ResolveGoProjectImports(
			absPath,
			filePath,
			b.ctx.dirToFiles,
			b.goPackageExportIndex,
			b.ctx.suppliedFiles,
			b.contentReader,
		)
	case ".kt":
		return kotlin.ResolveKotlinProjectImports(
			absPath,
			filePath,
			b.kotlinPackageIndex,
			b.kotlinPackageTypes,
			b.kotlinFilePackages,
			b.ctx.suppliedFiles,
			b.contentReader,
		)
	case ".ts", ".tsx":
		return typescript.ResolveTypeScriptProjectImports(absPath, filePath, ext, b.ctx.suppliedFiles, b.contentReader)
	default:
		return []string{}, nil
	}
}

func (b *defaultDependencyResolver) FinalizeGraph(graph DependencyGraph) error {
	return _go.AddGoIntraPackageDependencies(graph, b.ctx.goFiles, b.contentReader)
}
