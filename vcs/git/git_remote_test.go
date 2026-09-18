package git

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemoteOriginURL_ReturnsConfiguredOrigin(t *testing.T) {
	dir := t.TempDir()
	setupGitRepo(t, dir)

	cmd := exec.Command("git", "remote", "add", "origin", "git@example.com:foo/bar.git")
	cmd.Dir = dir
	require.NoError(t, cmd.Run())

	url, err := RemoteOriginURL(dir)
	require.NoError(t, err)
	assert.Equal(t, "git@example.com:foo/bar.git", url)
}

func TestRemoteOriginURL_NoRemoteConfigured_ReturnsEmptyNoError(t *testing.T) {
	dir := t.TempDir()
	setupGitRepo(t, dir)

	url, err := RemoteOriginURL(dir)
	require.NoError(t, err)
	assert.Empty(t, url)
}
