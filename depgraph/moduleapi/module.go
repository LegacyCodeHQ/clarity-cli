package moduleapi

import "github.com/LegacyCodeHQ/clarity/vcs"

// Module describes pluggable language support.
type Module interface {
	Name() string
	Extensions() []string
	Maturity() MaturityLevel
	NewResolver(ctx *Context, contentReader vcs.ContentReader) Resolver
	IsTestFile(filePath string, contentReader vcs.ContentReader) bool
}
