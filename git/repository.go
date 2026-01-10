package git

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// GetUncommittedDartFiles finds all uncommitted .dart files in a git repository.
// Returns absolute paths to all uncommitted files (staged, unstaged, and untracked).
func GetUncommittedDartFiles(repoPath string) ([]string, error) {
	// Validate the repository path exists
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("repository path does not exist: %s", repoPath)
	}

	// Verify it's a git repository
	if !isGitRepository(repoPath) {
		return nil, fmt.Errorf("%s is not a git repository (use 'git init' to initialize)", repoPath)
	}

	// Get the repository root
	repoRoot, err := getRepositoryRoot(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository root: %w", err)
	}

	// Get all uncommitted files
	uncommittedFiles, err := getUncommittedFiles(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get uncommitted files: %w", err)
	}

	// Filter for .dart files
	dartFiles := filterDartFiles(uncommittedFiles)

	// Convert to absolute paths
	absolutePaths := toAbsolutePaths(repoRoot, dartFiles)

	return absolutePaths, nil
}

// isGitRepository checks if the given path is inside a git repository
func isGitRepository(path string) bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = path
	err := cmd.Run()
	return err == nil
}

// getRepositoryRoot returns the absolute path to the repository root
func getRepositoryRoot(repoPath string) (string, error) {
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

// getUncommittedFiles returns a list of all uncommitted files (relative to repo root)
func getUncommittedFiles(repoPath string) ([]string, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = repoPath

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			// Check if git is not installed
			if strings.Contains(stderr.String(), "not found") || strings.Contains(stderr.String(), "not recognized") {
				return nil, fmt.Errorf("git command not found - please install Git to use the --repo flag")
			}
			return nil, fmt.Errorf("git command failed: %s", stderr.String())
		}
		return nil, err
	}

	// Parse the porcelain output
	var files []string
	lines := strings.Split(stdout.String(), "\n")
	for _, line := range lines {
		if len(line) < 3 {
			continue
		}

		// Porcelain format: XY filename
		// X = status in index, Y = status in working tree
		// We want all files that have any status
		filePath := strings.TrimSpace(line[3:])

		// Handle renamed files (format: "old -> new")
		if strings.Contains(filePath, " -> ") {
			parts := strings.Split(filePath, " -> ")
			filePath = parts[1] // Use the new filename
		}

		if filePath != "" {
			files = append(files, filePath)
		}
	}

	return files, nil
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
