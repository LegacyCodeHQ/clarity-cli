package typescript

import (
	"github.com/LegacyCodeHQ/clarity/depgraph/moduleapi"
	"github.com/LegacyCodeHQ/clarity/vcs"
)

type Provider struct{}

func (Provider) Name() string {
	return "TypeScript"
}

func (Provider) Extensions() []string {
	// .json and .sh aren't TypeScript — they ride along so package.json's
	// "scripts" field can be resolved to the shell scripts it invokes (see
	// package_scripts.go). ResolveProjectImports special-cases both.
	return []string{".ts", ".tsx", ".mts", ".cts", ".json", ".sh"}
}

func (Provider) Maturity() moduleapi.MaturityLevel {
	return moduleapi.MaturityBasicTests
}

func (Provider) NewResolver(ctx *moduleapi.Context, contentReader vcs.ContentReader) moduleapi.Resolver {
	return resolver{ctx: ctx, contentReader: contentReader}
}

func (Provider) IsTestFile(filePath string, _ vcs.ContentReader) bool {
	return IsTestFile(filePath)
}

type resolver struct {
	ctx           *moduleapi.Context
	contentReader vcs.ContentReader
}

func (r resolver) ResolveProjectImports(absPath, filePath, ext string) ([]string, error) {
	return ResolveTypeScriptProjectImports(absPath, filePath, ext, r.ctx.SuppliedFiles, r.contentReader)
}

func (resolver) FinalizeGraph(_ moduleapi.Graph) error {
	return nil
}
