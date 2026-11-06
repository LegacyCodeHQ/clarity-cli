package lean

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveLeanProjectImports(t *testing.T) {
	const source = "/project/src/MyLib/Main.lean"
	files := map[string]bool{
		source:                               true,
		"/project/src/MyLib/Basic.lean":      true,
		"/project/src/MyLib/with.dot.lean":   true,
		"/project/src/MyLib/α.lean":          true,
		"/other/Mathlib/Data/Nat/Basic.lean": true,
		"/project/src/Disabled.lean":         false,
	}
	content := []byte("import MyLib.Basic\nimport MyLib.«with.dot»\nimport MyLib.α\n" +
		"import MyLib.Main\nimport MyLib.Basic\nimport Mathlib.Data.Nat.Basic\nimport Disabled\n")
	// Only the importing file is readable. Resolution must use supplied files,
	// including when the reader represents a Git revision rather than disk.
	reader := func(path string) ([]byte, error) {
		if path == source {
			return content, nil
		}
		return nil, os.ErrNotExist
	}
	got, err := ResolveLeanProjectImports(source, files, reader)
	require.NoError(t, err)
	require.Equal(t, []string{
		"/project/src/MyLib/Basic.lean",
		"/project/src/MyLib/with.dot.lean",
		"/project/src/MyLib/α.lean",
	}, got)
}

func TestResolveLeanLakeImportsDoNotUseSiblingNames(t *testing.T) {
	config := "name = 'aesop'\n[[lean_lib]]\nname = 'Aesop'\n[[lean_lib]]\nname = 'AesopTest'\n"
	files := map[string]bool{
		"/project/Aesop.lean":           true,
		"/project/AesopTest/Aesop.lean": true,
	}
	reader := leanConfigReader(map[string]string{
		"/project/lakefile.toml":        config,
		"/project/AesopTest/Basic.lean": "import Aesop\n",
		"/project/AesopTest/Aesop.lean": "import Aesop\n",
	})
	for _, source := range []string{"/project/AesopTest/Basic.lean", "/project/AesopTest/Aesop.lean"} {
		got, err := ResolveLeanProjectImports(source, files, reader)
		require.NoError(t, err)
		require.Equal(t, []string{"/project/Aesop.lean"}, got)
	}
}

func TestResolveLeanLakeExternalImportsDoNotMatchNestedFiles(t *testing.T) {
	reader := leanConfigReader(map[string]string{
		"/project/lakefile.toml":                 "name = 'batteries'\n[[lean_lib]]\nname = 'Batteries'\n",
		"/project/Batteries/Lean/Meta/Expr.lean": "public import Lean.Expr\n",
	})
	got, err := ResolveLeanProjectImports("/project/Batteries/Lean/Meta/Expr.lean", map[string]bool{
		"/project/Batteries/Lean/Expr.lean": true,
	}, reader)
	require.NoError(t, err)
	require.Empty(t, got)
}

func TestResolveLeanLakeConfiguredSourceRoots(t *testing.T) {
	reader := leanConfigReader(map[string]string{
		"/project/lakefile.toml":     "name = 'sample'\nsrcDir = 'src'\n[[lean_lib]]\nname = 'Sample'\nsrcDir = 'lib'\n[[lean_exe]]\nname = 'app'\nsrcDir = 'bin'\n",
		"/project/src/bin/Main.lean": "import Sample.Basic\n",
	})
	got, err := ResolveLeanProjectImports("/project/src/bin/Main.lean", map[string]bool{
		"/project/src/lib/Sample/Basic.lean": true,
	}, reader)
	require.NoError(t, err)
	require.Equal(t, []string{"/project/src/lib/Sample/Basic.lean"}, got)
}

func TestResolveLeanLakeNestedPackageBoundary(t *testing.T) {
	reader := leanConfigReader(map[string]string{
		"/project/lakefile.toml":        "name = 'outer'",
		"/project/nested/lakefile.toml": "name = 'inner'",
		"/project/nested/Main.lean":     "import Outer\n",
	})
	got, err := ResolveLeanProjectImports("/project/nested/Main.lean", map[string]bool{
		"/project/Outer.lean": true,
	}, reader)
	require.NoError(t, err)
	require.Empty(t, got)
}

func TestResolveLeanLakeLeanConfigTakesPrecedenceOverTomlSourceRoots(t *testing.T) {
	reader := leanConfigReader(map[string]string{
		"/project/lakefile.lean": "import Lake\nopen Lake DSL\npackage sample\nlean_lib Lib\n",
		"/project/lakefile.toml": "name = 'sample'\nsrcDir = 'alternate'\n[[lean_lib]]\nname = 'Lib'\n",
		"/project/Main.lean":     "import Lib.Basic\n",
	})
	got, err := ResolveLeanProjectImports("/project/Main.lean", map[string]bool{
		"/project/Lib/Basic.lean":           true,
		"/project/alternate/Lib/Basic.lean": true,
	}, reader)
	require.NoError(t, err)
	require.Equal(t, []string{"/project/Lib/Basic.lean"}, got)
}

func TestResolveLeanProjectImportsPrefersNearestRoot(t *testing.T) {
	files := map[string]bool{
		"/project/MyLib/Basic.lean":        true,
		"/project/nested/MyLib/Basic.lean": true,
	}
	reader := leanConfigReader(map[string]string{
		"/project/nested/Main.lean": "import MyLib.Basic\n",
	})
	got, err := ResolveLeanProjectImports("/project/nested/Main.lean", files, reader)
	require.NoError(t, err)
	require.Equal(t, []string{"/project/nested/MyLib/Basic.lean"}, got)
}

func TestResolveLeanProjectImportsReadError(t *testing.T) {
	_, err := ResolveLeanProjectImports("/project/Main.lean", nil, func(string) ([]byte, error) {
		return nil, fmt.Errorf("unavailable revision")
	})
	require.ErrorContains(t, err, "unavailable revision")
}

func TestModulePathRejectsInvalidPaths(t *testing.T) {
	for _, name := range []string{"", "Foo..Bar", "«..»", "Foo.«../Bar»", "Foo.«a\\b»", "Foo.«unclosed"} {
		t.Run(name, func(t *testing.T) {
			_, ok := modulePath(name)
			require.False(t, ok)
		})
	}
}
