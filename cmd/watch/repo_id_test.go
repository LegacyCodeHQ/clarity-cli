package watch

import (
	"testing"

	"github.com/LegacyCodeHQ/clarity/cmd/watch/protocol"
	"github.com/stretchr/testify/assert"
)

func TestWorktreeIDForMain(t *testing.T) {
	assert.Equal(t, "main", worktreeIDFor("/any/path", protocol.WorktreeKindMain))
}

func TestWorktreeIDForLinkedIsStable(t *testing.T) {
	a := worktreeIDFor("/tmp/foo-feat", protocol.WorktreeKindLinked)
	b := worktreeIDFor("/tmp/foo-feat", protocol.WorktreeKindLinked)
	assert.Equal(t, a, b, "worktree id must be stable for the same path")
}

func TestWorktreeIDForLinkedHasPrefixAndLength(t *testing.T) {
	id := worktreeIDFor("/tmp/foo-feat", protocol.WorktreeKindLinked)
	assert.Contains(t, id, "wt-", "linked worktree id should be prefixed")
	assert.Equal(t, len("wt-")+8, len(id), "linked worktree id should be wt- + 8 hex chars")
}

func TestWorktreeIDForLinkedDistinguishesPaths(t *testing.T) {
	a := worktreeIDFor("/tmp/foo-feat-a", protocol.WorktreeKindLinked)
	b := worktreeIDFor("/tmp/foo-feat-b", protocol.WorktreeKindLinked)
	assert.NotEqual(t, a, b)
}

func TestPrimaryRepoLabel(t *testing.T) {
	cases := []struct {
		branch string
		want   string
	}{
		{"", "clarity-cli"},
		{"refs/heads/main", "main"},
		{"refs/heads/feat/foo", "feat/foo"},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, primaryRepoLabel("/Users/ragu/clarity-cli", c.branch), "branch=%q", c.branch)
	}
}

func TestLinkedRepoLabel(t *testing.T) {
	cases := []struct {
		path string
		want string
	}{
		{"/Users/ragu/clarity-cli", "clarity-cli"},
		{"/tmp/foo-feat", "foo-feat"},
		{"/", ""},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, linkedRepoLabel(c.path), "path=%q", c.path)
	}
}
