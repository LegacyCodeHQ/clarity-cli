package lean

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsTestFile(t *testing.T) {
	for _, path := range []string{"/project/tests/Basic.lean", "tests/Basic.lean", "/project/Tests/Basic.lean", "/project/test/Basic.lean", "Basic_test.lean", "test_basic.lean"} {
		require.True(t, IsTestFile(path), path)
	}
	for _, path := range []string{"/project/MyLib/Theorems.lean", "/project/MyLib/Examples.lean", "lakefile.lean", "test_basic.go"} {
		require.False(t, IsTestFile(path), path)
	}
}
