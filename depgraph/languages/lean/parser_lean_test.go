package lean

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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
