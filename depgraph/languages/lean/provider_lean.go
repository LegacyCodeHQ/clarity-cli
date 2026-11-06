package lean

import (
	"github.com/LegacyCodeHQ/clarity/depgraph/moduleapi"
	"github.com/LegacyCodeHQ/clarity/vcs"
)

type Provider struct{}

func (Provider) Name() string { return "Lean 4" }

func (Provider) Extensions() []string { return []string{".lean"} }

func (Provider) Maturity() moduleapi.MaturityLevel { return moduleapi.MaturityBasicTests }

func (Provider) NewResolver(ctx *moduleapi.Context, contentReader vcs.ContentReader) moduleapi.Resolver {
	lookup := &lakeLookup{reader: contentReader}
	return resolver{ctx: ctx, contentReader: contentReader, sourceRoots: lookup.sourceRoots}
}

func (Provider) IsTestFile(filePath string, contentReader vcs.ContentReader) bool {
	return IsTestFile(filePath) || isLakeTestFile(filePath, contentReader)
}

type resolver struct {
	ctx           *moduleapi.Context
	contentReader vcs.ContentReader
	sourceRoots   func(string) []string
}

func (r resolver) ResolveProjectImports(absPath, _, _ string) ([]string, error) {
	return resolveLeanProjectImports(absPath, r.ctx.SuppliedFiles, r.contentReader, r.sourceRoots)
}

func (resolver) FinalizeGraph(_ moduleapi.Graph) error { return nil }
