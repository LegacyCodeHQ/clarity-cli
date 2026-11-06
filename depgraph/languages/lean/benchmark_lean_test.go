package lean_test

import (
	"testing"

	"github.com/LegacyCodeHQ/clarity/depgraph/languages/lean"
	"github.com/LegacyCodeHQ/clarity/internal/testhelpers"
)

func BenchmarkLeanCorpus(b *testing.B) {
	testhelpers.BenchmarkLanguageCorpus(b, lean.Provider{}, "CLARITY_LEAN_BENCH_REPO")
}
