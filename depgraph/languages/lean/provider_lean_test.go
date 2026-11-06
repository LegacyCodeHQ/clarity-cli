package lean

import (
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/LegacyCodeHQ/clarity/depgraph/moduleapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLeanResolverCachesLakeDiscoveryConcurrently(t *testing.T) {
	contents := map[string]string{
		"/project/lakefile.toml":         "name = 'sample'\nsrcDir = 'src'\n",
		"/project/src/main.lean":         "import Lib.Basic\n",
		"/project/src/nested/other.lean": "import Lib.Basic\n",
	}
	var mu sync.Mutex
	reads := make(map[string]int)
	reader := func(path string) ([]byte, error) {
		mu.Lock()
		reads[path]++
		mu.Unlock()
		if source, ok := contents[path]; ok {
			return []byte(source), nil
		}
		return nil, os.ErrNotExist
	}
	r := (Provider{}).NewResolver(&moduleapi.Context{SuppliedFiles: map[string]bool{
		"/project/src/Lib/Basic.lean": true,
	}}, reader)
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			file := "/project/src/main.lean"
			if i%2 == 0 {
				file = "/project/src/nested/other.lean"
			}
			dependencies, err := r.ResolveProjectImports(file, file, ".lean")
			assert.NoError(t, err)
			assert.Equal(t, []string{"/project/src/Lib/Basic.lean"}, dependencies)
		}(i)
	}
	wg.Wait()
	for path, count := range reads {
		if strings.HasSuffix(path, "lakefile.lean") || strings.HasSuffix(path, "lakefile.toml") {
			require.Equal(t, 1, count, path)
		}
	}
}

func TestLeanResolverKeepsLakeSnapshotsSeparate(t *testing.T) {
	contents := map[string]string{
		"/project/lakefile.toml": "srcDir = 'before'",
		"/project/main.lean":     "import Lib.Basic\n",
	}
	reader := leanConfigReader(contents)
	ctx := &moduleapi.Context{SuppliedFiles: map[string]bool{
		"/project/before/Lib/Basic.lean": true,
		"/project/after/Lib/Basic.lean":  true,
	}}
	for _, directory := range []string{"before", "after"} {
		contents["/project/lakefile.toml"] = "srcDir = '" + directory + "'"
		r := (Provider{}).NewResolver(ctx, reader)
		dependencies, err := r.ResolveProjectImports("/project/main.lean", "main.lean", ".lean")
		require.NoError(t, err)
		require.Equal(t, []string{"/project/" + directory + "/Lib/Basic.lean"}, dependencies)
	}
}

func TestLeanResolverSkipsLakeDiscoveryWithoutImports(t *testing.T) {
	var reads []string
	_, err := ResolveLeanProjectImports("/project/main.lean", nil, func(path string) ([]byte, error) {
		reads = append(reads, path)
		return []byte("def greeting := \"hello\"\n"), nil
	})
	require.NoError(t, err)
	require.Equal(t, []string{"/project/main.lean"}, reads)
}
