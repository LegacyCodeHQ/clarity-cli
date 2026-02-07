package kotlin

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/LegacyCodeHQ/sanity/vcs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveKotlinSamePackageDependencies_IgnoresSelfDeclaredTopLevelTypes(t *testing.T) {
	tmpDir := t.TempDir()

	commonDir := filepath.Join(tmpDir, "src", "commonMain", "kotlin", "io", "github", "oshai", "kotlinlogging")
	jvmDir := filepath.Join(tmpDir, "src", "jvmMain", "kotlin", "io", "github", "oshai", "kotlinlogging")
	require.NoError(t, os.MkdirAll(commonDir, 0o755))
	require.NoError(t, os.MkdirAll(jvmDir, 0o755))

	commonConfig := filepath.Join(commonDir, "KotlinLoggingConfiguration.kt")
	jvmConfig := filepath.Join(jvmDir, "KotlinLoggingConfiguration.kt")
	levelFile := filepath.Join(commonDir, "Level.kt")

	require.NoError(t, os.WriteFile(commonConfig, []byte(`
package io.github.oshai.kotlinlogging

expect object KotlinLoggingConfiguration {
  var logLevel: Level
}
`), 0o644))
	require.NoError(t, os.WriteFile(jvmConfig, []byte(`
package io.github.oshai.kotlinlogging

actual object KotlinLoggingConfiguration {
  actual var logLevel: Level = Level.INFO
}
`), 0o644))
	require.NoError(t, os.WriteFile(levelFile, []byte(`
package io.github.oshai.kotlinlogging

enum class Level { INFO, DEBUG }
`), 0o644))

	contentReader := vcs.FilesystemContentReader()
	kotlinFiles := []string{commonConfig, jvmConfig, levelFile}
	packageIndex, packageTypes, filePackages := BuildKotlinIndices(kotlinFiles, contentReader)

	suppliedFiles := map[string]bool{
		commonConfig: true,
		jvmConfig:    true,
		levelFile:    true,
	}

	deps, err := ResolveKotlinProjectImports(
		jvmConfig,
		jvmConfig,
		packageIndex,
		packageTypes,
		filePackages,
		suppliedFiles,
		contentReader)
	require.NoError(t, err)
	assert.Contains(t, deps, levelFile)
	assert.NotContains(t, deps, commonConfig)
}
