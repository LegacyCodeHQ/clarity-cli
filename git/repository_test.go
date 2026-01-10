package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to setup a git repository in a temporary directory
func setupGitRepo(t *testing.T, dir string) {
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	require.NoError(t, cmd.Run(), "failed to initialize git repository")

	// Configure git user to avoid errors
	gitConfig(t, dir, "user.name", "Test User")
	gitConfig(t, dir, "user.email", "test@example.com")
}

// Helper function to set git config
func gitConfig(t *testing.T, repoDir, key, value string) {
	cmd := exec.Command("git", "config", key, value)
	cmd.Dir = repoDir
	require.NoError(t, cmd.Run(), "failed to set git config %s", key)
}

// Helper function to create a file with content
func createFile(t *testing.T, dir, name, content string) string {
	filePath := filepath.Join(dir, name)
	err := os.WriteFile(filePath, []byte(content), 0644)
	require.NoError(t, err, "failed to create file %s", name)
	return filePath
}

// Helper function to create a dart file
func createDartFile(t *testing.T, dir, name string) string {
	return createFile(t, dir, name, "import 'dart:io';\n\nclass Test {}")
}

// Helper function to add a file to git staging area
func gitAdd(t *testing.T, repoDir, file string) {
	cmd := exec.Command("git", "add", file)
	cmd.Dir = repoDir
	require.NoError(t, cmd.Run(), "failed to git add %s", file)
}

// Helper function to commit files
func gitCommit(t *testing.T, repoDir, message string) {
	cmd := exec.Command("git", "commit", "-m", message)
	cmd.Dir = repoDir
	require.NoError(t, cmd.Run(), "failed to git commit")
}

// Helper function to modify a file
func modifyFile(t *testing.T, filePath string) {
	err := os.WriteFile(filePath, []byte("modified content\n"), 0644)
	require.NoError(t, err, "failed to modify file %s", filePath)
}

func TestGetUncommittedDartFiles_UntrackedFiles(t *testing.T) {
	tmpDir := t.TempDir()
	setupGitRepo(t, tmpDir)

	// Create untracked .dart files
	dartFile1 := createDartFile(t, tmpDir, "test1.dart")
	dartFile2 := createDartFile(t, tmpDir, "test2.dart")

	// Get uncommitted files
	files, err := GetUncommittedDartFiles(tmpDir)

	require.NoError(t, err)
	assert.Len(t, files, 2)

	// Resolve symlinks for comparison (macOS /var -> /private/var)
	resolved1, _ := filepath.EvalSymlinks(dartFile1)
	resolved2, _ := filepath.EvalSymlinks(dartFile2)
	assert.Contains(t, files, resolved1)
	assert.Contains(t, files, resolved2)
}

func TestGetUncommittedDartFiles_StagedFiles(t *testing.T) {
	tmpDir := t.TempDir()
	setupGitRepo(t, tmpDir)

	// Create and stage .dart files
	dartFile := createDartFile(t, tmpDir, "test.dart")
	gitAdd(t, tmpDir, "test.dart")

	// Get uncommitted files
	files, err := GetUncommittedDartFiles(tmpDir)

	require.NoError(t, err)
	assert.Len(t, files, 1)
	resolved, _ := filepath.EvalSymlinks(dartFile)
	assert.Contains(t, files, resolved)
}

func TestGetUncommittedDartFiles_ModifiedFiles(t *testing.T) {
	tmpDir := t.TempDir()
	setupGitRepo(t, tmpDir)

	// Create, commit, then modify a .dart file
	dartFile := createDartFile(t, tmpDir, "test.dart")
	gitAdd(t, tmpDir, "test.dart")
	gitCommit(t, tmpDir, "Initial commit")

	// Modify the file
	modifyFile(t, dartFile)

	// Get uncommitted files
	files, err := GetUncommittedDartFiles(tmpDir)

	require.NoError(t, err)
	assert.Len(t, files, 1)
	resolved, _ := filepath.EvalSymlinks(dartFile)
	assert.Contains(t, files, resolved)
}

func TestGetUncommittedDartFiles_MixedStates(t *testing.T) {
	tmpDir := t.TempDir()
	setupGitRepo(t, tmpDir)

	// Create a committed file
	committedFile := createDartFile(t, tmpDir, "committed.dart")
	gitAdd(t, tmpDir, "committed.dart")
	gitCommit(t, tmpDir, "Initial commit")

	// Create a staged file
	stagedFile := createDartFile(t, tmpDir, "staged.dart")
	gitAdd(t, tmpDir, "staged.dart")

	// Create an untracked file
	untrackedFile := createDartFile(t, tmpDir, "untracked.dart")

	// Modify the committed file
	modifyFile(t, committedFile)

	// Get uncommitted files
	files, err := GetUncommittedDartFiles(tmpDir)

	require.NoError(t, err)
	assert.Len(t, files, 3)
	resolved1, _ := filepath.EvalSymlinks(committedFile)
	resolved2, _ := filepath.EvalSymlinks(stagedFile)
	resolved3, _ := filepath.EvalSymlinks(untrackedFile)
	assert.Contains(t, files, resolved1)  // modified
	assert.Contains(t, files, resolved2)  // staged
	assert.Contains(t, files, resolved3)  // untracked
}

func TestGetUncommittedDartFiles_FiltersDartOnly(t *testing.T) {
	tmpDir := t.TempDir()
	setupGitRepo(t, tmpDir)

	// Create various file types
	dartFile := createDartFile(t, tmpDir, "test.dart")
	createFile(t, tmpDir, "test.go", "package main")
	createFile(t, tmpDir, "README.md", "# Test")
	createFile(t, tmpDir, "test.txt", "text file")

	// Get uncommitted files
	files, err := GetUncommittedDartFiles(tmpDir)

	require.NoError(t, err)
	assert.Len(t, files, 1, "should only include .dart files")
	resolved, _ := filepath.EvalSymlinks(dartFile)
	assert.Contains(t, files, resolved)
}

func TestGetUncommittedDartFiles_NoUncommittedFiles(t *testing.T) {
	tmpDir := t.TempDir()
	setupGitRepo(t, tmpDir)

	// Create and commit a file
	createDartFile(t, tmpDir, "test.dart")
	gitAdd(t, tmpDir, "test.dart")
	gitCommit(t, tmpDir, "Initial commit")

	// Get uncommitted files (should be empty)
	files, err := GetUncommittedDartFiles(tmpDir)

	require.NoError(t, err)
	assert.Empty(t, files)
}

func TestGetUncommittedDartFiles_EmptyRepo(t *testing.T) {
	tmpDir := t.TempDir()
	setupGitRepo(t, tmpDir)

	// Get uncommitted files from empty repo
	files, err := GetUncommittedDartFiles(tmpDir)

	require.NoError(t, err)
	assert.Empty(t, files)
}

func TestGetUncommittedDartFiles_NotGitRepo(t *testing.T) {
	tmpDir := t.TempDir()
	// Don't initialize git

	// Try to get uncommitted files
	_, err := GetUncommittedDartFiles(tmpDir)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not a git repository")
}

func TestGetUncommittedDartFiles_InvalidPath(t *testing.T) {
	_, err := GetUncommittedDartFiles("/nonexistent/path")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestIsGitRepository_Valid(t *testing.T) {
	tmpDir := t.TempDir()
	setupGitRepo(t, tmpDir)

	isRepo := isGitRepository(tmpDir)

	assert.True(t, isRepo)
}

func TestIsGitRepository_Invalid(t *testing.T) {
	tmpDir := t.TempDir()
	// Don't initialize git

	isRepo := isGitRepository(tmpDir)

	assert.False(t, isRepo)
}

func TestFilterDartFiles(t *testing.T) {
	files := []string{
		"test1.dart",
		"test2.go",
		"README.md",
		"test3.dart",
		"script.sh",
	}

	dartFiles := filterDartFiles(files)

	assert.Len(t, dartFiles, 2)
	assert.Contains(t, dartFiles, "test1.dart")
	assert.Contains(t, dartFiles, "test3.dart")
}

func TestToAbsolutePaths(t *testing.T) {
	repoRoot := "/Users/test/repo"
	relativePaths := []string{
		"lib/main.dart",
		"test/widget_test.dart",
		"models/user.dart",
	}

	absolutePaths := toAbsolutePaths(repoRoot, relativePaths)

	assert.Len(t, absolutePaths, 3)
	assert.Equal(t, "/Users/test/repo/lib/main.dart", absolutePaths[0])
	assert.Equal(t, "/Users/test/repo/test/widget_test.dart", absolutePaths[1])
	assert.Equal(t, "/Users/test/repo/models/user.dart", absolutePaths[2])
}
