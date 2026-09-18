package git

import (
	"path/filepath"
	"strings"
)

// WorktreeKind classifies a worktree using git's own vocabulary: the single
// main worktree created by `git init`/`git clone`, or one of possibly several
// linked worktrees created via `git worktree add`. See git-worktree(1).
type WorktreeKind string

const (
	WorktreeKindMain   WorktreeKind = "main"
	WorktreeKindLinked WorktreeKind = "linked"
)

// Worktree describes a single working tree registered with a repository.
type Worktree struct {
	// Path is the absolute path to the working tree on disk.
	Path string
	// Head is the SHA at HEAD, or empty for an unborn branch.
	Head string
	// Branch is the full ref name (e.g. "refs/heads/main"), empty for detached.
	Branch string
	// Kind is WorktreeKindMain for the repository's main worktree,
	// WorktreeKindLinked otherwise. Per git's `worktree list --porcelain`
	// contract, the main worktree is always first.
	Kind WorktreeKind
}

// GetGitDir returns the absolute path to the git directory for `path`.
// For the primary worktree this equals GetCommonDir; for a linked worktree
// this is the per-worktree gitdir under `<common>/worktrees/<name>`.
func GetGitDir(path string) (string, error) {
	stdout, stderr, err := runGitCommand(path, "rev-parse", "--absolute-git-dir")
	if err != nil {
		return "", gitCommandError(err, stderr)
	}
	return strings.TrimSpace(string(stdout)), nil
}

// GetCommonDir returns the absolute path to the common git directory — the
// primary repository's `.git`. Identical for the primary and every linked
// worktree of the same repository.
func GetCommonDir(path string) (string, error) {
	stdout, stderr, err := runGitCommand(path, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return "", gitCommandError(err, stderr)
	}
	return strings.TrimSpace(string(stdout)), nil
}

// WorktreeKindFor reports whether `path` is inside the main worktree of its
// repository or a linked one. It's WorktreeKindMain iff the git dir equals
// the common dir.
func WorktreeKindFor(path string) (WorktreeKind, error) {
	gitDir, err := GetGitDir(path)
	if err != nil {
		return "", err
	}
	commonDir, err := GetCommonDir(path)
	if err != nil {
		return "", err
	}
	if resolveSymlinks(gitDir) == resolveSymlinks(commonDir) {
		return WorktreeKindMain, nil
	}
	return WorktreeKindLinked, nil
}

// ListWorktrees enumerates all worktrees registered with the repository
// containing `path`. The main worktree is always the first entry.
func ListWorktrees(path string) ([]Worktree, error) {
	stdout, stderr, err := runGitCommand(path, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, gitCommandError(err, stderr)
	}
	return parseWorktreePorcelain(string(stdout)), nil
}

// parseWorktreePorcelain parses the output of `git worktree list --porcelain`.
// Stanzas are separated by blank lines. Each stanza begins with `worktree <path>`,
// optionally followed by `HEAD <sha>` and either `branch <ref>`, `detached`, or
// `bare`. The first stanza describes the main worktree.
func parseWorktreePorcelain(out string) []Worktree {
	var (
		result   []Worktree
		current  Worktree
		hasEntry bool
	)
	flush := func() {
		if hasEntry {
			if len(result) == 0 {
				current.Kind = WorktreeKindMain
			} else {
				current.Kind = WorktreeKindLinked
			}
			result = append(result, current)
		}
		current = Worktree{}
		hasEntry = false
	}
	for _, line := range strings.Split(out, "\n") {
		trimmed := strings.TrimRight(line, "\r")
		if trimmed == "" {
			flush()
			continue
		}
		key, value, _ := strings.Cut(trimmed, " ")
		switch key {
		case "worktree":
			flush()
			current.Path = filepath.Clean(value)
			hasEntry = true
		case "HEAD":
			current.Head = value
		case "branch":
			current.Branch = value
		case "detached", "bare", "locked":
			// no-op for our purposes
		}
	}
	flush()
	return result
}
