package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseGoImports_BasicImports(t *testing.T) {
	source := `
package main

import "fmt"
import "os"
import "io"

func main() {
	fmt.Println("Hello")
}
`
	imports, err := ParseGoImports([]byte(source))

	require.NoError(t, err)
	assert.Len(t, imports, 3)

	assert.Contains(t, imports, StandardLibraryImport{"fmt"})
	assert.Contains(t, imports, StandardLibraryImport{"os"})
	assert.Contains(t, imports, StandardLibraryImport{"io"})
}

func TestParseGoImports_MultipleImports(t *testing.T) {
	source := `
package main

import (
	"fmt"
	"os"
	"net/http"
	"encoding/json"
)

func main() {}
`
	imports, err := ParseGoImports([]byte(source))

	require.NoError(t, err)
	assert.Len(t, imports, 4)

	assert.Contains(t, imports, StandardLibraryImport{"fmt"})
	assert.Contains(t, imports, StandardLibraryImport{"os"})
	assert.Contains(t, imports, StandardLibraryImport{"net/http"})
	assert.Contains(t, imports, StandardLibraryImport{"encoding/json"})
}

func TestParseGoImports_ExternalImports(t *testing.T) {
	source := `
package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func main() {}
`
	imports, err := ParseGoImports([]byte(source))

	require.NoError(t, err)
	assert.Len(t, imports, 3)

	assert.Contains(t, imports, StandardLibraryImport{"fmt"})
	assert.Contains(t, imports, ExternalImport{"github.com/spf13/cobra"})
	assert.Contains(t, imports, ExternalImport{"github.com/stretchr/testify/assert"})
}

func TestParseGoImports_InternalImports(t *testing.T) {
	source := `
package main

import (
	"fmt"
	"sanity/parser"
	"sanity/cmd"
)

func main() {}
`
	imports, err := ParseGoImports([]byte(source))

	require.NoError(t, err)
	assert.Len(t, imports, 3)

	assert.Contains(t, imports, StandardLibraryImport{"fmt"})
	assert.Contains(t, imports, InternalImport{"sanity/parser"})
	assert.Contains(t, imports, InternalImport{"sanity/cmd"})
}

func TestParseGoImports_AliasedImports(t *testing.T) {
	source := `
package main

import (
	f "fmt"
	h "net/http"
	. "os"
)

func main() {}
`
	imports, err := ParseGoImports([]byte(source))

	require.NoError(t, err)
	assert.Len(t, imports, 3)

	assert.Contains(t, imports, StandardLibraryImport{"fmt"})
	assert.Contains(t, imports, StandardLibraryImport{"net/http"})
	assert.Contains(t, imports, StandardLibraryImport{"os"})
}

func TestParseGoImports_BlankImports(t *testing.T) {
	source := `
package main

import (
	"fmt"
	_ "database/sql"
	_ "github.com/lib/pq"
)

func main() {}
`
	imports, err := ParseGoImports([]byte(source))

	require.NoError(t, err)
	assert.Len(t, imports, 3)

	assert.Contains(t, imports, StandardLibraryImport{"fmt"})
	assert.Contains(t, imports, StandardLibraryImport{"database/sql"})
	assert.Contains(t, imports, ExternalImport{"github.com/lib/pq"})
}

func TestParseGoImports_EmptyFile(t *testing.T) {
	source := ``
	imports, err := ParseGoImports([]byte(source))

	require.NoError(t, err)
	assert.Empty(t, imports)
}

func TestParseGoImports_NoImports(t *testing.T) {
	source := `
package main

func main() {
	println("No imports here")
}
`
	imports, err := ParseGoImports([]byte(source))

	require.NoError(t, err)
	assert.Empty(t, imports)
}

func TestParseGoImports_InvalidGoCode(t *testing.T) {
	source := `
this is not valid go code @#$%^
`
	// Should not panic, might return empty or error
	imports, err := ParseGoImports([]byte(source))

	// Either error or empty result is acceptable
	if err == nil {
		assert.NotNil(t, imports)
	}
}

func TestGoImports_FileNotFound(t *testing.T) {
	_, err := GoImports("/nonexistent/file/path.go")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read file")
}

func TestGoImports_ValidFile(t *testing.T) {
	// Create a temporary Go file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.go")

	content := `
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("Hello, World!")
}
`
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	require.NoError(t, err)

	// Extract imports
	imports, err := GoImports(tmpFile)

	require.NoError(t, err)
	assert.Len(t, imports, 2)

	assert.Contains(t, imports, StandardLibraryImport{"fmt"})
	assert.Contains(t, imports, StandardLibraryImport{"os"})
}

func TestParseGoImports_ComplexExample(t *testing.T) {
	source := `
// This is a comment
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/smacker/go-tree-sitter"
	"github.com/spf13/cobra"

	"sanity/parser"
	"sanity/cmd"
)

func main() {
	fmt.Println("Complex example")
}
`
	imports, err := ParseGoImports([]byte(source))

	require.NoError(t, err)
	assert.Len(t, imports, 8)

	// Standard library imports
	assert.Contains(t, imports, StandardLibraryImport{"context"})
	assert.Contains(t, imports, StandardLibraryImport{"fmt"})
	assert.Contains(t, imports, StandardLibraryImport{"net/http"})
	assert.Contains(t, imports, StandardLibraryImport{"os"})

	// External imports
	assert.Contains(t, imports, ExternalImport{"github.com/smacker/go-tree-sitter"})
	assert.Contains(t, imports, ExternalImport{"github.com/spf13/cobra"})

	// Internal imports
	assert.Contains(t, imports, InternalImport{"sanity/parser"})
	assert.Contains(t, imports, InternalImport{"sanity/cmd"})
}

func TestParseGoImports_MixedStandardAndExternal(t *testing.T) {
	source := `
package main

import (
	"context"
	"fmt"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"
)

func main() {}
`
	imports, err := ParseGoImports([]byte(source))

	require.NoError(t, err)
	assert.Len(t, imports, 4)

	assert.Contains(t, imports, StandardLibraryImport{"context"})
	assert.Contains(t, imports, StandardLibraryImport{"fmt"})
	assert.Contains(t, imports, ExternalImport{"github.com/smacker/go-tree-sitter"})
	assert.Contains(t, imports, ExternalImport{"github.com/smacker/go-tree-sitter/golang"})
}
