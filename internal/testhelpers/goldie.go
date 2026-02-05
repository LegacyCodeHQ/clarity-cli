package testhelpers

import (
	"testing"

	"github.com/sebdah/goldie/v2"
)

// GoldieWithSuffix creates a Goldie instance with a golden file suffix.
func GoldieWithSuffix(t *testing.T, suffix string) *goldie.Goldie {
	t.Helper()
	return goldie.New(t, goldie.WithNameSuffix(suffix))
}
