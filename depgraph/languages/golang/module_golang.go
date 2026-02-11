package golang

import (
	"errors"

	"github.com/LegacyCodeHQ/clarity/depgraph/moduleapi"
	"github.com/LegacyCodeHQ/clarity/vcs"
	graphlib "github.com/dominikbraun/graph"
)

type Module struct{}

func (Module) Name() string {
	return "Go"
}

func (Module) Extensions() []string {
	return []string{".go"}
}

func (Module) Maturity() moduleapi.MaturityLevel {
	return moduleapi.MaturityActivelyTested
}

func (Module) NewResolver(ctx *moduleapi.Context, contentReader vcs.ContentReader) moduleapi.Resolver {
	return resolver{
		ctx:             ctx,
		contentReader:   contentReader,
		projectResolver: NewProjectImportResolver(ctx.DirToFiles, ctx.SuppliedFiles, contentReader),
	}
}

func (Module) IsTestFile(filePath string, _ vcs.ContentReader) bool {
	return IsTestFile(filePath)
}

type resolver struct {
	ctx             *moduleapi.Context
	contentReader   vcs.ContentReader
	projectResolver *ProjectImportResolver
}

func (r resolver) ResolveProjectImports(absPath, filePath, _ string) ([]string, error) {
	return r.projectResolver.ResolveProjectImports(absPath, filePath)
}

func (r resolver) FinalizeGraph(graph moduleapi.Graph) error {
	return addGoIntraPackageDependencies(graph, r.ctx.GoFiles, r.contentReader)
}

func addGoIntraPackageDependencies(
	graph moduleapi.Graph,
	goFiles []string,
	contentReader vcs.ContentReader,
) error {
	if len(goFiles) == 0 {
		return nil
	}

	intraDeps, err := BuildIntraPackageDependencies(goFiles, vcs.ContentReader(contentReader))
	if err != nil {
		return err
	}

	for file, deps := range intraDeps {
		if _, err := graph.Vertex(file); err != nil {
			continue
		}
		for _, dep := range deps {
			if _, err := graph.Vertex(dep); err != nil {
				continue
			}
			if err := graph.AddEdge(file, dep); err != nil && !errors.Is(err, graphlib.ErrEdgeAlreadyExists) {
				return err
			}
		}
	}

	return nil
}
