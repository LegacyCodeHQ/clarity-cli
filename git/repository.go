package git

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// FileStats represents statistics for a file (additions and deletions)
type FileStats struct {
	Additions int
	Deletions int
}

// GetUncommittedDartFiles finds all uncommitted files in a git repository.
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
	repoRoot, err := GetRepositoryRoot(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository root: %w", err)
	}

	// Get all uncommitted files
	uncommittedFiles, err := getUncommittedFiles(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get uncommitted files: %w", err)
	}

	// Convert to absolute paths (no filtering - include all files)
	absolutePaths := toAbsolutePaths(repoRoot, uncommittedFiles)

	return absolutePaths, nil
}

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

// GetCommitDartFiles finds all files that were changed in a specific commit.
// Returns absolute paths to all files added, modified, or renamed in the commit.
func GetCommitDartFiles(repoPath, commitID string) ([]string, error) {
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

	// Get files changed in the commit
	commitFiles, err := getCommitFiles(repoPath, commitID)
	if err != nil {
		return nil, fmt.Errorf("failed to get files from commit: %w", err)
	}

	// Convert to absolute paths (no filtering - include all files)
	absolutePaths := toAbsolutePaths(repoRoot, commitFiles)

	return absolutePaths, nil
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

// getCommitFiles returns a list of all files changed in the specified commit (relative to repo root)
func getCommitFiles(repoPath, commitID string) ([]string, error) {
	// Use --root flag to handle root commits (first commit in repo)
	// Use --diff-filter=d to exclude deleted files (only include added, modified, and renamed files)
	cmd := exec.Command("git", "diff-tree", "--no-commit-id", "--name-only", "-r", "--root", "--diff-filter=d", commitID)
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

	return files, nil
}

// GetFileContentFromCommit reads the content of a file at a specific commit
// using 'git show commit:path'. The filePath should be relative to the repository root.
func GetFileContentFromCommit(repoPath, commitID, filePath string) ([]byte, error) {
	// Format: commit:path
	ref := fmt.Sprintf("%s:%s", commitID, filePath)

	cmd := exec.Command("git", "show", ref)
	cmd.Dir = repoPath

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return nil, fmt.Errorf("git show failed: %s", stderr.String())
		}
		return nil, err
	}

	return stdout.Bytes(), nil
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

// GetUncommittedFileStats returns statistics (additions/deletions) for uncommitted files
// Returns a map from relative file paths to their FileStats
func GetUncommittedFileStats(repoPath string) (map[string]FileStats, error) {
	// Validate the repository path exists
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("repository path does not exist: %s", repoPath)
	}

	// Verify it's a git repository
	if !isGitRepository(repoPath) {
		return nil, fmt.Errorf("%s is not a git repository", repoPath)
	}

	// Get the repository root
	repoRoot, err := GetRepositoryRoot(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository root: %w", err)
	}

	// Run git diff --numstat to get stats for uncommitted changes
	// This includes both staged and unstaged changes
	cmd := exec.Command("git", "diff", "--numstat", "HEAD")
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

	// Parse the numstat output
	stats := make(map[string]FileStats)
	lines := strings.Split(stdout.String(), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Format: additions	deletions	filename
		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}

		additions := 0
		deletions := 0

		// Parse additions (may be "-" for binary files)
		if parts[0] != "-" {
			additions, _ = strconv.Atoi(parts[0])
		}

		// Parse deletions (may be "-" for binary files)
		if parts[1] != "-" {
			deletions, _ = strconv.Atoi(parts[1])
		}

		// Handle renamed files (format: "old => new")
		filePath := strings.Join(parts[2:], " ")
		if strings.Contains(filePath, " => ") {
			renameParts := strings.Split(filePath, " => ")
			filePath = strings.TrimSpace(renameParts[1])
		}

		// Convert to absolute path
		absPath := filepath.Join(repoRoot, filePath)

		stats[absPath] = FileStats{
			Additions: additions,
			Deletions: deletions,
		}
	}

	return stats, nil
}

// GetCommitFileStats returns statistics (additions/deletions) for files in a specific commit
// Returns a map from absolute file paths to their FileStats
func GetCommitFileStats(repoPath, commitID string) (map[string]FileStats, error) {
	// Validate the repository path exists
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("repository path does not exist: %s", repoPath)
	}

	// Verify it's a git repository
	if !isGitRepository(repoPath) {
		return nil, fmt.Errorf("%s is not a git repository", repoPath)
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

	// Run git show --numstat to get stats for the commit
	// Use --root flag to handle root commits
	cmd := exec.Command("git", "show", "--numstat", "--format=", commitID)
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

	// Parse the numstat output
	stats := make(map[string]FileStats)
	lines := strings.Split(stdout.String(), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Format: additions	deletions	filename
		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}

		additions := 0
		deletions := 0

		// Parse additions (may be "-" for binary files)
		if parts[0] != "-" {
			additions, _ = strconv.Atoi(parts[0])
		}

		// Parse deletions (may be "-" for binary files)
		if parts[1] != "-" {
			deletions, _ = strconv.Atoi(parts[1])
		}

		// Handle renamed files (format: "old => new")
		filePath := strings.Join(parts[2:], " ")
		if strings.Contains(filePath, " => ") {
			renameParts := strings.Split(filePath, " => ")
			filePath = strings.TrimSpace(renameParts[1])
		}

		// Convert to absolute path
		absPath := filepath.Join(repoRoot, filePath)

		stats[absPath] = FileStats{
			Additions: additions,
			Deletions: deletions,
		}
	}

	return stats, nil
}
