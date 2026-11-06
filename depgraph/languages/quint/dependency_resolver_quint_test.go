package quint

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveQuintProjectImports(t *testing.T) {
	const source = "/project/specs/main.qnt"
	reader := func(path string) ([]byte, error) {
		require.Equal(t, source, path)
		return []byte(`module main {
  import Local.* from "./local"
  import Local as Alias from "./folder/../local"
  import Shared(X = 2).* from "../shared"
  import Self from "./main"
  import External from "./external"
  import LocalWithoutFrom.*
}`), nil
	}
	got, err := ResolveQuintProjectImports(source, map[string]bool{
		source:                                true,
		"/project/specs/local.qnt":            true,
		"/project/shared.qnt":                 true,
		"/project/specs/external.qnt":         false,
		"/project/specs/LocalWithoutFrom.qnt": true,
	}, reader)
	require.NoError(t, err)
	require.Equal(t, []string{"/project/specs/local.qnt", "/project/shared.qnt"}, got)
}

func TestResolveQuintAppendsExtensionLikeCompiler(t *testing.T) {
	got, err := ResolveQuintProjectImports("/project/main.qnt", map[string]bool{
		"/project/library.qnt":     true,
		"/project/library.qnt.qnt": true,
	}, func(string) ([]byte, error) {
		return []byte(`module main { import Lib.* from "./library.qnt" }`), nil
	})
	require.NoError(t, err)
	require.Equal(t, []string{"/project/library.qnt.qnt"}, got)
}

func TestResolveQuintLiteralBackslash(t *testing.T) {
	path := filepath.Join("/project", `folder\name.qnt`)
	got, err := ResolveQuintProjectImports("/project/main.qnt", map[string]bool{path: true}, func(string) ([]byte, error) {
		return []byte(`module main { import Lib.* from "./folder\name" }`), nil
	})
	require.NoError(t, err)
	require.Equal(t, []string{path}, got)
}

func TestResolveQuintReadError(t *testing.T) {
	_, err := ResolveQuintProjectImports("/project/main.qnt", nil, func(string) ([]byte, error) {
		return nil, os.ErrNotExist
	})
	require.ErrorIs(t, err, os.ErrNotExist)
}
