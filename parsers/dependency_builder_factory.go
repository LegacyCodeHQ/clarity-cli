package parsers

import (
	"github.com/LegacyCodeHQ/sanity/parsers/dart"
	_go "github.com/LegacyCodeHQ/sanity/parsers/go"
	"github.com/LegacyCodeHQ/sanity/parsers/kotlin"
	"github.com/LegacyCodeHQ/sanity/parsers/typescript"
	"github.com/LegacyCodeHQ/sanity/vcs"
)

// DependencyBuilder builds project imports per file and can finalize graph-wide dependencies.
type DependencyBuilder interface {
	BuildProjectImports(absPath, filePath, ext string) ([]string, error)
	FinalizeGraph(graph DependencyGraph) error
}

type dependencyBuilderFactory struct {
	ctx                  *dependencyGraphContext
	contentReader        vcs.ContentReader
	goPackageExportIndex map[string]_go.GoPackageExportIndex
	kotlinPackageIndex   map[string][]string
	kotlinPackageTypes   map[string]map[string][]string
	kotlinFilePackages   map[string]string
}

// NewDependencyBuilder creates a language-aware dependency builder using precomputed indices.
func NewDependencyBuilder(ctx *dependencyGraphContext, contentReader vcs.ContentReader) DependencyBuilder {
	goPackageExportIndex := _go.BuildGoPackageExportIndices(ctx.dirToFiles, contentReader)
	kotlinPackageIndex, kotlinPackageTypes, kotlinFilePackages := kotlin.BuildKotlinIndices(ctx.kotlinFiles, contentReader)

	return &dependencyBuilderFactory{
		ctx:                  ctx,
		contentReader:        contentReader,
		goPackageExportIndex: goPackageExportIndex,
		kotlinPackageIndex:   kotlinPackageIndex,
		kotlinPackageTypes:   kotlinPackageTypes,
		kotlinFilePackages:   kotlinFilePackages,
	}
}

func (f *dependencyBuilderFactory) BuildProjectImports(absPath, filePath, ext string) ([]string, error) {
	switch ext {
	case ".dart":
		return dart.BuildDartProjectImports(absPath, filePath, ext, f.ctx.suppliedFiles, f.contentReader)
	case ".go":
		return _go.BuildGoProjectImports(
			absPath,
			filePath,
			f.ctx.dirToFiles,
			f.goPackageExportIndex,
			f.ctx.suppliedFiles,
			f.contentReader,
		)
	case ".kt":
		return kotlin.BuildKotlinProjectImports(
			absPath,
			filePath,
			f.kotlinPackageIndex,
			f.kotlinPackageTypes,
			f.kotlinFilePackages,
			f.ctx.suppliedFiles,
			f.contentReader,
		)
	case ".ts", ".tsx":
		return typescript.BuildTypeScriptProjectImports(absPath, filePath, ext, f.ctx.suppliedFiles, f.contentReader)
	default:
		return []string{}, nil
	}
}

func (f *dependencyBuilderFactory) FinalizeGraph(graph DependencyGraph) error {
	return _go.AddGoIntraPackageDependencies(graph, f.ctx.goFiles, f.contentReader)
}

