package lean

import (
	"os"
	"testing"

	"github.com/LegacyCodeHQ/clarity/vcs"
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

func TestLakeTestDriverClassification(t *testing.T) {
	const config = `name = "dxn"
testDriver = "DxnTest"
[[lean_lib]]
name = "Dxn"
[[lean_lib]]
name = "DxnTest"
`
	provider := Provider{}
	reader := leanConfigReader(map[string]string{"/project/lakefile.toml": config})
	for _, path := range []string{"/project/DxnTest.lean", "/project/DxnTest/Name.lean"} {
		require.True(t, provider.IsTestFile(path, reader), path)
	}
	for _, path := range []string{"/project/Dxn.lean", "/project/Dxn/Name.lean", "/project/DxnSpec.lean", "/project/DxnTestExtra.lean", "/project/DxnTest/Name.ts"} {
		require.False(t, provider.IsTestFile(path, reader), path)
	}
}

func TestLakeTestDriverRoots(t *testing.T) {
	tests := []struct {
		name   string
		config string
		path   string
		want   bool
	}{
		{"custom library roots", "testDriver = 'check'\n[[lean_lib]]\nname = 'check'\nroots = ['Verify.Core', 'Verify.Extra']", "/project/Verify/Core/Name.lean", true},
		{"quoted module root", "testDriver = 'check'\n[[lean_lib]]\nname = 'check'\nroots = ['Verify.«with.dot»']", "/project/Verify/with.dot/Name.lean", true},
		{"custom root overrides name", "testDriver = 'check'\n[[lean_lib]]\nname = 'check'\nroots = ['Verify.Core']", "/project/check.lean", false},
		{"empty roots", "testDriver = 'check'\n[[lean_lib]]\nname = 'check'\nroots = []", "/project/check.lean", false},
		{"package and target source directories", "testDriver = 'check'\nsrcDir = 'src'\n[[lean_lib]]\nname = 'check'\nsrcDir = 'verification'\nroots = ['Verify']", "/project/src/verification/Verify/Name.lean", true},
		{"outside configured source directory", "testDriver = 'check'\nsrcDir = 'src'\n[[lean_lib]]\nname = 'check'", "/project/check.lean", false},
		{"executable root", "testDriver = 'check'\n[[lean_exe]]\nname = 'check'\nroot = 'Verify.Main'", "/project/Verify/Main.lean", true},
		{"executable does not own a namespace", "testDriver = 'check'\n[[lean_exe]]\nname = 'check'\nroot = 'Verify.Main'", "/project/Verify/Main/Production.lean", false},
		{"executable default root", "testDriver = 'check'\n[[lean_exe]]\nname = 'check'", "/project/check.lean", true},
		{"missing driver target", "testDriver = 'check'\n[[lean_lib]]\nname = 'Other'", "/project/check.lean", false},
		{"no configured driver", "[[lean_lib]]\nname = 'DxnTest'", "/project/DxnTest.lean", false},
		{"invalid config", "testDriver = 'check'\n[[lean_lib]\nname = 'check'", "/project/check.lean", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := leanConfigReader(map[string]string{"/project/lakefile.toml": tt.config})
			require.Equal(t, tt.want, (Provider{}).IsTestFile(tt.path, reader))
		})
	}
}

func TestLakeTestDriverPackageBoundaries(t *testing.T) {
	parent := "testDriver = 'DxnTest'\n[[lean_lib]]\nname = 'DxnTest'\n"
	for _, nested := range []map[string]string{
		{"/project/DxnTest/nested/lakefile.toml": "name = 'nested'"},
		{"/project/DxnTest/nested/lakefile.toml": "invalid TOML"},
		{"/project/DxnTest/nested/lakefile.lean": "import Lake\nopen Lake DSL\npackage nested"},
	} {
		nested["/project/lakefile.toml"] = parent
		require.False(t, (Provider{}).IsTestFile("/project/DxnTest/nested/Main.lean", leanConfigReader(nested)))
	}
	require.False(t, (Provider{}).IsTestFile("/project/DxnTest.lean", nil))
	require.True(t, (Provider{}).IsTestFile("/project/tests/Basic.lean", nil))
}

func TestLakeLeanConfigTakesPrecedenceOverTomlTestDriver(t *testing.T) {
	reader := leanConfigReader(map[string]string{
		"/project/lakefile.lean": "import Lake\nopen Lake DSL\npackage sample\nlean_lib Lib\n",
		"/project/lakefile.toml": "testDriver = 'WrongTest'\n[[lean_lib]]\nname = 'WrongTest'\n",
	})
	require.False(t, (Provider{}).IsTestFile("/project/WrongTest/Basic.lean", reader))
}

func leanConfigReader(files map[string]string) vcs.ContentReader {
	return func(path string) ([]byte, error) {
		if content, ok := files[path]; ok {
			return []byte(content), nil
		}
		return nil, os.ErrNotExist
	}
}
