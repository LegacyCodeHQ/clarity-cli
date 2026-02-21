package formatters

import "strings"

// GraphDirection represents the direction of the graph layout.
type GraphDirection string

const (
	DirectionLR GraphDirection = "LR"
	DirectionRL GraphDirection = "RL"
	DirectionTB GraphDirection = "TB"
	DirectionBT GraphDirection = "BT"
)

// DefaultDirection is the default layout direction for graphs.
const DefaultDirection = DirectionLR

// String returns the canonical string representation.
func (d GraphDirection) String() string {
	return string(d)
}

// StringLower returns the canonical string representation in lowercase.
func (d GraphDirection) StringLower() string {
	return strings.ToLower(string(d))
}

// ParseDirection converts a string to GraphDirection.
func ParseDirection(s string) (GraphDirection, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "lr":
		return DirectionLR, true
	case "rl":
		return DirectionRL, true
	case "tb":
		return DirectionTB, true
	case "bt":
		return DirectionBT, true
	default:
		return DefaultDirection, false
	}
}

// SupportedDirections returns a list of all supported direction names.
func SupportedDirections() string {
	return "lr, rl, tb, bt"
}
