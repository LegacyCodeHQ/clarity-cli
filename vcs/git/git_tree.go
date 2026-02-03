package git

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// GetCommitTreeFiles returns all files that exist in a commit's tree.
// Unlike GetCommitDartFiles which only returns files changed in a commit,
// this returns all files that existed at that point in time.
// Returns absolute paths to all files in the commit tree.
func GetCommitTreeFiles(repoPath, commitID string) ([]string, error) {
	// Validate the repository path exists
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("repository path does not exist: %s", repoPath)
	}

	// Verify it's a git repository
	if !isGitRepository(repoPath) {
		return nil, fmt.Errorf("%s is not a git repository (use 'git init' to initialize)", repoPath)
	}

	// Validate the commit exists
	if err := validateCommit(repoPath, commitID); err != nil {
		return nil, err
	}

	// Get the repository root
	repoRoot, err := GetRepositoryRoot(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository root: %w", err)
	}

	// Use git ls-tree to list all files in the commit tree
	cmd := exec.Command("git", "ls-tree", "-r", "--name-only", commitID)
	cmd.Dir = repoPath

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return nil, fmt.Errorf("git command failed: %s", stderr.String())
		}
		return nil, err
	}

	// Parse the output - one file per line
	var files []string
	lines := strings.Split(stdout.String(), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			files = append(files, line)
		}
	}

	// Convert to absolute paths
	absolutePaths := toAbsolutePaths(repoRoot, files)

	return absolutePaths, nil
}
