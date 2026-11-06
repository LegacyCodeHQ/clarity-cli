package quint

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
			name: "all file import forms",
			source: `module main {
  import Wild.* from "./wild"
  import Named.name from "./named"
  import Mod as M from "./module"
  import Proto(X = 1) as P from "./instance"
  import Other(X = 1).* from "./anonymous"
}`,
			want: []string{"./wild", "./named", "./module", "./instance", "./anonymous"},
		},
		{
			name: "multiple modules and duplicates",
			source: `module first { import A.* from "../shared/a" }
module second { import A as Alias from "../shared/a"
  import B from "./b"
}`,
			want: []string{"../shared/a", "./b"},
		},
		{
			name:   "literal string contents",
			source: `module main { import A.* from "./folder\name" }`,
			want:   []string{`./folder\name`},
		},
		{
			name: "comments and unrelated strings",
			source: `// import Fake from "./line"
/* import Fake from "./block" */
module main {
  /// import Fake from "./doc"
  pure val text = "from ./string"
  import A.* from /* comment */ "./a"
}`,
			want: []string{"./a"},
		},
		{
			name: "module imports and exports have no implicit file path",
			source: `module A { pure val value = 1 }
module main { import A.*
  import A as Alias
  export A.*
}`,
		},
		{
			name:   "hashbang and CRLF",
			source: "#!/usr/bin/env quint\r\nmodule main {\r\nimport A.* from \"./a\"\r\n}\r\n",
			want:   []string{"./a"},
		},
		{name: "empty"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseImports([]byte(tt.source))
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
