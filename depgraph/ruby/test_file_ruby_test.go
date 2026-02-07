package ruby

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsTestFile(t *testing.T) {
	assert.True(t, IsTestFile("spec/user_spec.rb"))
	assert.True(t, IsTestFile("test/user_test.rb"))
	assert.False(t, IsTestFile("lib/user.rb"))
}
