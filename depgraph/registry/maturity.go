package registry

import "github.com/LegacyCodeHQ/clarity/depgraph/moduleapi"

// MaturityLevel describes how complete a language's analysis support is.
type MaturityLevel = moduleapi.MaturityLevel

const (
	MaturityUntested       = moduleapi.MaturityUntested
	MaturityBasicTests     = moduleapi.MaturityBasicTests
	MaturityActivelyTested = moduleapi.MaturityActivelyTested
	MaturityStable         = moduleapi.MaturityStable
)

// MaturityLevels returns the ordered set of known maturity levels.
func MaturityLevels() []MaturityLevel {
	return moduleapi.MaturityLevels()
}
