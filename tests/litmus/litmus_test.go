package litmus

import (
	"testing"

	"github.com/sebdah/goldie/v2"
	"github.com/stretchr/testify/assert"
)

func TestEnvironmentIsSetup(t *testing.T) {
	actual := 2 + 2
	expected := 4

	assert.Equal(t, expected, actual)
}

func TestGoldieIsSetup(t *testing.T) {
	g := goldie.New(t)
	g.Assert(t, t.Name(), []byte("Goldie is setup!"))
}
