package ruby

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRubyImports(t *testing.T) {
	source := `
require "json"
require_relative "../lib/core"
require('set')
# require "ignored"
`

	imports, err := ParseRubyImports([]byte(source))

	require.NoError(t, err)
	assert.Len(t, imports, 3)
	assert.Equal(t, "json", imports[0].Path())
	assert.False(t, imports[0].IsRelative())
	assert.Equal(t, "../lib/core", imports[1].Path())
	assert.True(t, imports[1].IsRelative())
	assert.Equal(t, "set", imports[2].Path())
}

func TestRubyImports_ValidFile(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "app.rb")

	err := os.WriteFile(tmpFile, []byte("require 'json'\n"), 0644)
	require.NoError(t, err)

	imports, err := RubyImports(tmpFile)

	require.NoError(t, err)
	assert.Len(t, imports, 1)
	assert.Equal(t, "json", imports[0].Path())
}

func TestResolveRubyImportPath(t *testing.T) {
	suppliedFiles := map[string]bool{
		"/project/lib/core.rb":        true,
		"/project/app/models/user.rb": true,
	}

	rel := ResolveRubyImportPath("/project/app/main.rb", RubyImport{path: "../lib/core", isRelative: true}, suppliedFiles)
	assert.Equal(t, []string{"/project/lib/core.rb"}, rel)

	abs := ResolveRubyImportPath("/project/app/main.rb", RubyImport{path: "app/models/user", isRelative: false}, suppliedFiles)
	assert.Equal(t, []string{"/project/app/models/user.rb"}, abs)
}

func TestParseRubyConstantReferences(t *testing.T) {
	source := `
class CacheCoderTest < ActiveSupport::TestCase
  setup do
    @coder = ActiveSupport::Cache::Coder.new(Serializer)
    @error = ::JSON::ParserError
  end
end
`

	refs := ParseRubyConstantReferences([]byte(source))
	assert.ElementsMatch(t, []string{
		"ActiveSupport::TestCase",
		"ActiveSupport::Cache::Coder",
		"JSON::ParserError",
	}, refs)
}

func TestResolveRubyConstantReferencePath(t *testing.T) {
	suppliedFiles := map[string]bool{
		"/project/activesupport/lib/active_support/cache/coder.rb": true,
		"/project/activesupport/lib/active_support/cache/entry.rb": true,
	}

	resolved := ResolveRubyConstantReferencePath("ActiveSupport::Cache::Coder", suppliedFiles)
	assert.Equal(t, []string{"/project/activesupport/lib/active_support/cache/coder.rb"}, resolved)
}

func TestResolveRubyConstantReferencePath_Ambiguous(t *testing.T) {
	suppliedFiles := map[string]bool{
		"/project/a/lib/active_support/cache/coder.rb": true,
		"/project/b/lib/active_support/cache/coder.rb": true,
	}

	resolved := ResolveRubyConstantReferencePath("ActiveSupport::Cache::Coder", suppliedFiles)
	assert.Empty(t, resolved)
}

func TestResolveRubyConstantReferencePath_AllowsIntermediateDirectories(t *testing.T) {
	suppliedFiles := map[string]bool{
		"/project/actionpack/lib/action_controller/metal/request_forgery_protection.rb": true,
	}

	resolved := ResolveRubyConstantReferencePath("ActionController::RequestForgeryProtection", suppliedFiles)
	assert.Equal(t, []string{"/project/actionpack/lib/action_controller/metal/request_forgery_protection.rb"}, resolved)
}

func TestResolveRubyConstantReferencePath_PrefersMoreSpecificPath(t *testing.T) {
	suppliedFiles := map[string]bool{
		"/project/lib/action_controller/request_forgery_protection.rb":                  true,
		"/project/actionpack/lib/action_controller/metal/request_forgery_protection.rb": true,
	}

	resolved := ResolveRubyConstantReferencePath("ActionController::RequestForgeryProtection", suppliedFiles)
	assert.Equal(t, []string{"/project/lib/action_controller/request_forgery_protection.rb"}, resolved)
}

func TestResolveRubyConstantReferencePath_UsesEnclosingConstantPrefix(t *testing.T) {
	suppliedFiles := map[string]bool{
		"/project/actionpack/lib/action_controller/metal/request_forgery_protection.rb": true,
	}

	resolved := ResolveRubyConstantReferencePath("ActionController::RequestForgeryProtection::AUTHENTICITY_TOKEN_LENGTH", suppliedFiles)
	assert.Equal(t, []string{"/project/actionpack/lib/action_controller/metal/request_forgery_protection.rb"}, resolved)
}
