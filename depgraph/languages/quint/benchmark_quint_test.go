package quint_test

import (
	"testing"

	"github.com/LegacyCodeHQ/clarity/depgraph/languages/quint"
	"github.com/LegacyCodeHQ/clarity/internal/testhelpers"
)

func BenchmarkQuintCorpus(b *testing.B) {
	testhelpers.BenchmarkLanguageCorpus(b, quint.Provider{}, "CLARITY_QUINT_BENCH_REPO")
}
