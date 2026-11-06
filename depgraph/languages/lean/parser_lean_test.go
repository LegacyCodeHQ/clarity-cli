package lean

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseManyLeanImports(t *testing.T) {
	var source strings.Builder
	var expected []string
	for i := 0; i < 300; i++ {
		name := fmt.Sprintf("Library.Module%d", i)
		fmt.Fprintf(&source, "import %s\n", name)
		expected = append(expected, name)
	}
	got, err := ParseImports([]byte(source.String()))
	require.NoError(t, err)
	require.Equal(t, expected, got)
}

func TestLeanImportsSurviveErrorRoot(t *testing.T) {
	// A partial nested do block can make tree-sitter recover with an ERROR root
	// instead of module. The valid file header must still contribute imports.
	source := []byte("import Library.Basic\ndef inspect : IO (Array Nat) := do\n" +
		"  let (first?, second?, third?) ← readValues?\n" +
		"  let result ← input.values.flatMapM fun value => do")
	got, err := ParseImports(source)
	require.NoError(t, err)
	require.Equal(t, []string{"Library.Basic"}, got)
}

func TestParseImports(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   []string
	}{
		{
			name: "ordinary imports and duplicates",
			source: "import MyLib.Basic\nimport Mathlib.Data.Nat.Basic\nimport MyLib.Basic\n" +
				"theorem identity (n : Nat) : n = n := by\n  rfl\n",
			want: []string{"MyLib.Basic", "Mathlib.Data.Nat.Basic"},
		},
		{
			name: "module modifiers and Unicode identifiers",
			source: "module\npublic import MyLib.Basic\nmeta import Lean\n" +
				"public meta import all MyLib.Internal\nimport MyLib.«with.dot»\nimport MyLib.α\n",
			want: []string{"MyLib.Basic", "Lean", "MyLib.Internal", "MyLib.«with.dot»", "MyLib.α"},
		},
		{
			name: "comments and strings are not imports",
			source: "/-! import Fake.Doc -/\n/- import Fake.Block /- import Fake.Nested -/ -/\n" +
				"-- import Fake.Line\nimport /- comment -/ MyLib.Basic\n" +
				"def text := \"import Fake.String\"\n",
			want: []string{"MyLib.Basic"},
		},
		{
			name: "imports survive unsupported body syntax",
			source: "import MyLib.Basic\nsyntax \"custom\" : term\n" +
				"def broken := @@@ custom\n",
			want: []string{"MyLib.Basic"},
		},
		{
			name: "header comments and quoted comment markers",
			source: "module\nprelude\npublic /- outer /- nested -/ -/ meta import all MyLib.«not--a/-comment»\n" +
				"import Other-- adjacent line comment\nimport Third/- adjacent block comment -/\n" +
				"public meta section\ndef text := \"import Fake.Body\"\n",
			want: []string{"MyLib.«not--a/-comment»", "Other", "Third"},
		},
		{
			name:   "quoted identifiers containing spaces and opening guillemets",
			source: "import MyLib.«with space»\nimport MyLib.«left«angle»\ndef f := 1\n",
			want:   []string{"MyLib.«with space»", "MyLib.«left«angle»"},
		},
		{
			name:   "unfinished header retains complete preceding imports",
			source: "import MyLib.Basic\nimport /- unfinished",
			want:   []string{"MyLib.Basic"},
		},
		{
			name:   "import text occurs only in declarations",
			source: "def text := \"import Fake.Body\"\n",
		},
		{name: "empty"},
		{name: "prelude", source: "prelude\ndef x := 1\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseImports([]byte(tt.source))
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
