package java

import (
	"github.com/LegacyCodeHQ/clarity/depgraph/moduleapi"
	"github.com/LegacyCodeHQ/clarity/vcs"
)

type Module struct{}

func (Module) Name() string {
	return "Java"
}

func (Module) Extensions() []string {
	return []string{".java"}
}

func (Module) Maturity() moduleapi.MaturityLevel {
	return moduleapi.MaturityBasicTests
}

func (Module) NewResolver(ctx *moduleapi.Context, contentReader vcs.ContentReader) moduleapi.Resolver {
	packageIndex, packageTypes, filePackages := BuildJavaIndices(ctx.JavaFiles, contentReader)
	return resolver{
		ctx:           ctx,
		contentReader: contentReader,
		packageIndex:  packageIndex,
		packageTypes:  packageTypes,
		filePackages:  filePackages,
	}
}

func (Module) IsTestFile(filePath string, _ vcs.ContentReader) bool {
	return IsTestFile(filePath)
}

type resolver struct {
	ctx           *moduleapi.Context
	contentReader vcs.ContentReader
	packageIndex  map[string][]string
	packageTypes  map[string]map[string][]string
	filePackages  map[string]string
}

func (r resolver) ResolveProjectImports(absPath, filePath, _ string) ([]string, error) {
	return ResolveJavaProjectImports(
		absPath,
		filePath,
		r.packageIndex,
		r.packageTypes,
		r.filePackages,
		r.ctx.SuppliedFiles,
		r.contentReader)
}

func (resolver) FinalizeGraph(_ moduleapi.Graph) error {
	return nil
}
