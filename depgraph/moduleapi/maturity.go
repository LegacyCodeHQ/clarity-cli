package moduleapi

// MaturityLevel describes how complete a language's analysis support is.
type MaturityLevel int

const (
	MaturityUntested MaturityLevel = iota
	MaturityBasicTests
	MaturityActivelyTested
	MaturityStable
)

func (level MaturityLevel) DisplayName() string {
	switch level {
	case MaturityUntested:
		return "Untested"
	case MaturityBasicTests:
		return "Basic Tests"
	case MaturityActivelyTested:
		return "Actively Tested"
	case MaturityStable:
		return "Stable"
	default:
		return "Unknown"
	}
}

func (level MaturityLevel) Symbol() string {
	switch level {
	case MaturityUntested:
		return "○"
	case MaturityBasicTests:
		return "◐"
	case MaturityActivelyTested:
		return "●"
	case MaturityStable:
		return "✓"
	default:
		return "?"
	}
}

// MaturityLevels returns the ordered set of known maturity levels.
func MaturityLevels() []MaturityLevel {
	return []MaturityLevel{
		MaturityUntested,
		MaturityBasicTests,
		MaturityActivelyTested,
		MaturityStable,
	}
}
