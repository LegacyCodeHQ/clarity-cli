package lean

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveLeanQuotedModulePreservesOpeningGuillemet(t *testing.T) {
	reader := leanConfigReader(map[string]string{
		"/project/lakefile.toml": "name = 'sample'",
		"/project/Main.lean":     "import MyLib.«left«angle»\n",
	})
	got, err := ResolveLeanProjectImports("/project/Main.lean", map[string]bool{
		"/project/MyLib/left«angle.lean": true,
	}, reader)
	require.NoError(t, err)
	require.Equal(t, []string{"/project/MyLib/left«angle.lean"}, got)
}
