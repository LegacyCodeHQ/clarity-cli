package store

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveRepoOrigin_WithRemote_ReturnsRemoteURL(t *testing.T) {
	repoPath := initRepoWithCommit(t)
	cmd := exec.Command("git", "remote", "add", "origin", "git@example.com:foo/bar.git")
	cmd.Dir = repoPath
	require.NoError(t, cmd.Run())

	origin, err := ResolveRepoOrigin(repoPath)
	require.NoError(t, err)
	assert.Equal(t, "git@example.com:foo/bar.git", origin)
}

func TestResolveRepoOrigin_NoRemote_FallsBackToCommonDir(t *testing.T) {
	repoPath := initRepoWithCommit(t)

	origin, err := ResolveRepoOrigin(repoPath)
	require.NoError(t, err)
	assert.NotEmpty(t, origin)
	assert.NotContains(t, origin, "http", "the fallback identity must not look like a real remote URL")
}

func TestEnsureProject_FirstCall_CreatesRow(t *testing.T) {
	db := openMigratedTestDB(t)

	id, err := EnsureProject(db, "origin-a")
	require.NoError(t, err)
	assert.NotEmpty(t, id)

	var count int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM projects WHERE repo_origin = ?`, "origin-a").Scan(&count))
	assert.Equal(t, 1, count)
}

func TestEnsureProject_RepeatedCalls_ReuseSameRowNoDuplicate(t *testing.T) {
	db := openMigratedTestDB(t)

	first, err := EnsureProject(db, "origin-a")
	require.NoError(t, err)

	second, err := EnsureProject(db, "origin-a")
	require.NoError(t, err)

	assert.Equal(t, first, second, "the same repo_origin must resolve to the same project id")

	var count int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM projects WHERE repo_origin = ?`, "origin-a").Scan(&count))
	assert.Equal(t, 1, count, "must never insert a duplicate row for the same repo_origin")
}

func TestEnsureProject_DifferentOrigins_GetDifferentRows(t *testing.T) {
	db := openMigratedTestDB(t)

	a, err := EnsureProject(db, "origin-a")
	require.NoError(t, err)
	b, err := EnsureProject(db, "origin-b")
	require.NoError(t, err)

	assert.NotEqual(t, a, b)
}
