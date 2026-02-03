package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// isGitRepository checks if the given path is inside a git repository
func isGitRepository(path string) bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = path
	err := cmd.Run()
	return err == nil
}

// GetRepositoryRoot returns the absolute path to the repository root
func GetRepositoryRoot(repoPath string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = repoPath

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return "", fmt.Errorf("git command failed: %s", stderr.String())
		}
		return "", err
	}

	return strings.TrimSpace(stdout.String()), nil
}

// filterDartFiles filters a list of file paths to include only .dart files
func filterDartFiles(files []string) []string {
	var dartFiles []string
	for _, file := range files {
		if filepath.Ext(file) == ".dart" {
			dartFiles = append(dartFiles, file)
		}
	}
	return dartFiles
}

// toAbsolutePaths converts relative paths to absolute paths based on the repository root
func toAbsolutePaths(repoRoot string, relativePaths []string) []string {
	var absolutePaths []string
	for _, relPath := range relativePaths {
		absPath := filepath.Join(repoRoot, relPath)
		absolutePaths = append(absolutePaths, absPath)
	}
	return absolutePaths
}

// validateCommit checks if the given commit reference exists in the repository
func validateCommit(repoPath, commitID string) error {
	cmd := exec.Command("git", "rev-parse", "--verify", commitID+"^{commit}")
	cmd.Dir = repoPath

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return fmt.Errorf("invalid commit reference '%s': %s", commitID, strings.TrimSpace(stderr.String()))
		}
		return fmt.Errorf("invalid commit reference '%s'", commitID)
	}

	return nil
}

// GetCurrentCommitHash returns the current commit hash (HEAD)
func GetCurrentCommitHash(repoPath string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	cmd.Dir = repoPath

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return "", fmt.Errorf("git command failed: %s", stderr.String())
		}
		return "", err
	}

	return strings.TrimSpace(stdout.String()), nil
}

// GetShortCommitHash returns the short version of a given commit hash
func GetShortCommitHash(repoPath, commitID string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--short", commitID)
	cmd.Dir = repoPath

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return "", fmt.Errorf("git command failed: %s", stderr.String())
		}
		return "", err
	}

	return strings.TrimSpace(stdout.String()), nil
}

// HasUncommittedChanges checks if there are any uncommitted changes in the repository
func HasUncommittedChanges(repoPath string) (bool, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = repoPath

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return false, fmt.Errorf("git command failed: %s", stderr.String())
		}
		return false, err
	}

	// If output is not empty, there are uncommitted changes
	return strings.TrimSpace(stdout.String()) != "", nil
}

// ParseCommitRange parses a commit specification and returns the from/to commits.
// Supports formats: "abc...def", "abc..def", or single commit "abc"
// Returns (from, to, isRange)
func ParseCommitRange(commitSpec string) (string, string, bool) {
	// Check for three-dot syntax first (more specific)
	if strings.Contains(commitSpec, "...") {
		parts := strings.SplitN(commitSpec, "...", 2)
		return parts[0], parts[1], true
	}
	// Check for two-dot syntax
	if strings.Contains(commitSpec, "..") {
		parts := strings.SplitN(commitSpec, "..", 2)
		return parts[0], parts[1], true
	}
	// Single commit
	return "", commitSpec, false
}

// isAncestor checks if possibleAncestor is an ancestor of possibleDescendant.
// Returns true if possibleAncestor is older than (or equal to) possibleDescendant.
func isAncestor(repoPath, possibleAncestor, possibleDescendant string) (bool, error) {
	cmd := exec.Command("git", "merge-base", "--is-ancestor", possibleAncestor, possibleDescendant)
	cmd.Dir = repoPath
	err := cmd.Run()
	if err != nil {
		// Exit code 1 means not an ancestor, which is not an error for our purposes
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// NormalizeCommitRange ensures commits are in chronological order (older first).
// If the commits are reversed (newer...older), it swaps them.
// Returns (olderCommit, newerCommit, swapped, error)
func NormalizeCommitRange(repoPath, from, to string) (string, string, bool, error) {
	// Check if 'from' is an ancestor of 'to' (correct order)
	isCorrectOrder, err := isAncestor(repoPath, from, to)
	if err != nil {
		return from, to, false, err
	}

	if isCorrectOrder {
		// Already in correct order (from is older)
		return from, to, false, nil
	}

	// Check if 'to' is an ancestor of 'from' (reversed order)
	isReversed, err := isAncestor(repoPath, to, from)
	if err != nil {
		return from, to, false, err
	}

	if isReversed {
		// Commits are reversed, swap them
		return to, from, true, nil
	}

	// Commits are not in a linear ancestry (e.g., different branches)
	// Keep original order - git diff will still work
	return from, to, false, nil
}

// GetCommitRangeLabel returns a label like "abc123...def456" for display
func GetCommitRangeLabel(repoPath, fromCommit, toCommit string) (string, error) {
	fromShort, err := GetShortCommitHash(repoPath, fromCommit)
	if err != nil {
		return "", err
	}
	toShort, err := GetShortCommitHash(repoPath, toCommit)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s...%s", fromShort, toShort), nil
}
