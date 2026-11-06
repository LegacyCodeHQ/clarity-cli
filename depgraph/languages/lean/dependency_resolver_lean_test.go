package lean

import (
	"fmt"
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
		require.Equal(t, source, path)
		return content, nil
	}
	got, err := ResolveLeanProjectImports(source, files, reader)
	require.NoError(t, err)
	require.Equal(t, []string{
		"/project/src/MyLib/Basic.lean",
		"/project/src/MyLib/with.dot.lean",
		"/project/src/MyLib/α.lean",
	}, got)
}

func TestResolveLeanProjectImportsPrefersNearestRoot(t *testing.T) {
	files := map[string]bool{
		"/project/MyLib/Basic.lean":        true,
		"/project/nested/MyLib/Basic.lean": true,
	}
	got, err := ResolveLeanProjectImports("/project/nested/Main.lean", files, func(string) ([]byte, error) {
		return []byte("import MyLib.Basic\n"), nil
	})
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
