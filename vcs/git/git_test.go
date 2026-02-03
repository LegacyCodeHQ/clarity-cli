package git

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

// Tests for GetRepositoryRoot

func TestGetRepositoryRoot_FromRepoRoot(t *testing.T) {
	tmpDir := t.TempDir()
	setupGitRepo(t, tmpDir)

	root, err := GetRepositoryRoot(tmpDir)

	require.NoError(t, err)
	// Resolve symlinks for comparison (macOS /var -> /private/var)
	resolvedTmp, _ := filepath.EvalSymlinks(tmpDir)
	assert.Equal(t, resolvedTmp, root)
}

func TestGetRepositoryRoot_FromSubdirectory(t *testing.T) {
	tmpDir := t.TempDir()
	setupGitRepo(t, tmpDir)

	// Create subdirectory
	subDir := filepath.Join(tmpDir, "lib", "src")
	err := os.MkdirAll(subDir, 0755)
	require.NoError(t, err)

	root, err := GetRepositoryRoot(subDir)

	require.NoError(t, err)
	resolvedTmp, _ := filepath.EvalSymlinks(tmpDir)
	assert.Equal(t, resolvedTmp, root)
}

func TestGetRepositoryRoot_NotGitRepo(t *testing.T) {
	tmpDir := t.TempDir()
	// Don't initialize git

	_, err := GetRepositoryRoot(tmpDir)

	assert.Error(t, err)
}

// Tests for GetCurrentCommitHash

func TestGetCurrentCommitHash_Success(t *testing.T) {
	tmpDir := t.TempDir()
	setupGitRepo(t, tmpDir)

	createFile(t, tmpDir, "test.txt", "content")
	gitAdd(t, tmpDir, "test.txt")
	gitCommit(t, tmpDir, "Initial commit")

	hash, err := GetCurrentCommitHash(tmpDir)

	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	// Short hash is typically 7 characters
	assert.True(t, len(hash) >= 7 && len(hash) <= 12, "hash should be short format")
}

func TestGetCurrentCommitHash_MatchesHEAD(t *testing.T) {
	tmpDir := t.TempDir()
	setupGitRepo(t, tmpDir)

	createFile(t, tmpDir, "test.txt", "content")
	gitAdd(t, tmpDir, "test.txt")
	expectedHash := gitCommitAndGetSHA(t, tmpDir, "Initial commit")

	hash, err := GetCurrentCommitHash(tmpDir)

	require.NoError(t, err)
	// The current hash should be a prefix of the full commit SHA
	assert.True(t, strings.HasPrefix(expectedHash, hash), "current hash should match HEAD")
}

func TestGetCurrentCommitHash_NoCommits(t *testing.T) {
	tmpDir := t.TempDir()
	setupGitRepo(t, tmpDir)

	// No commits made
	_, err := GetCurrentCommitHash(tmpDir)

	assert.Error(t, err)
}

// Tests for GetShortCommitHash

func TestGetShortCommitHash_Success(t *testing.T) {
	tmpDir := t.TempDir()
	setupGitRepo(t, tmpDir)

	createFile(t, tmpDir, "test.txt", "content")
	gitAdd(t, tmpDir, "test.txt")
	fullHash := gitCommitAndGetSHA(t, tmpDir, "Initial commit")

	shortHash, err := GetShortCommitHash(tmpDir, fullHash)

	require.NoError(t, err)
	assert.True(t, len(shortHash) >= 7 && len(shortHash) <= 12)
	assert.True(t, strings.HasPrefix(fullHash, shortHash))
}

func TestGetShortCommitHash_AlreadyShort(t *testing.T) {
	tmpDir := t.TempDir()
	setupGitRepo(t, tmpDir)

	createFile(t, tmpDir, "test.txt", "content")
	gitAdd(t, tmpDir, "test.txt")
	fullHash := gitCommitAndGetSHA(t, tmpDir, "Initial commit")

	// Get short hash first
	shortHash, err := GetShortCommitHash(tmpDir, fullHash[:7])

	require.NoError(t, err)
	assert.NotEmpty(t, shortHash)
}

func TestGetShortCommitHash_HEAD(t *testing.T) {
	tmpDir := t.TempDir()
	setupGitRepo(t, tmpDir)

	createFile(t, tmpDir, "test.txt", "content")
	gitAdd(t, tmpDir, "test.txt")
	gitCommit(t, tmpDir, "Initial commit")

	shortHash, err := GetShortCommitHash(tmpDir, "HEAD")

	require.NoError(t, err)
	assert.NotEmpty(t, shortHash)
}

func TestGetShortCommitHash_InvalidCommit(t *testing.T) {
	tmpDir := t.TempDir()
	setupGitRepo(t, tmpDir)

	createFile(t, tmpDir, "test.txt", "content")
	gitAdd(t, tmpDir, "test.txt")
	gitCommit(t, tmpDir, "Initial commit")

	_, err := GetShortCommitHash(tmpDir, "invalid-sha-that-does-not-exist")

	assert.Error(t, err)
}

// Tests for HasUncommittedChanges

func TestHasUncommittedChanges_Clean(t *testing.T) {
	tmpDir := t.TempDir()
	setupGitRepo(t, tmpDir)

	createFile(t, tmpDir, "test.txt", "content")
	gitAdd(t, tmpDir, "test.txt")
	gitCommit(t, tmpDir, "Initial commit")

	hasChanges, err := HasUncommittedChanges(tmpDir)

	require.NoError(t, err)
	assert.False(t, hasChanges)
}

func TestHasUncommittedChanges_UntrackedFile(t *testing.T) {
	tmpDir := t.TempDir()
	setupGitRepo(t, tmpDir)

	createFile(t, tmpDir, "committed.txt", "content")
	gitAdd(t, tmpDir, "committed.txt")
	gitCommit(t, tmpDir, "Initial commit")

	// Create untracked file
	createFile(t, tmpDir, "untracked.txt", "new content")

	hasChanges, err := HasUncommittedChanges(tmpDir)

	require.NoError(t, err)
	assert.True(t, hasChanges)
}

func TestHasUncommittedChanges_ModifiedFile(t *testing.T) {
	tmpDir := t.TempDir()
	setupGitRepo(t, tmpDir)

	filePath := createFile(t, tmpDir, "test.txt", "content")
	gitAdd(t, tmpDir, "test.txt")
	gitCommit(t, tmpDir, "Initial commit")

	// Modify the file
	modifyFile(t, filePath)

	hasChanges, err := HasUncommittedChanges(tmpDir)

	require.NoError(t, err)
	assert.True(t, hasChanges)
}

func TestHasUncommittedChanges_StagedFile(t *testing.T) {
	tmpDir := t.TempDir()
	setupGitRepo(t, tmpDir)

	createFile(t, tmpDir, "first.txt", "content")
	gitAdd(t, tmpDir, "first.txt")
	gitCommit(t, tmpDir, "Initial commit")

	// Create and stage a new file
	createFile(t, tmpDir, "staged.txt", "new content")
	gitAdd(t, tmpDir, "staged.txt")

	hasChanges, err := HasUncommittedChanges(tmpDir)

	require.NoError(t, err)
	assert.True(t, hasChanges)
}

func TestHasUncommittedChanges_EmptyRepo(t *testing.T) {
	tmpDir := t.TempDir()
	setupGitRepo(t, tmpDir)

	// Empty repo with no commits
	hasChanges, err := HasUncommittedChanges(tmpDir)

	require.NoError(t, err)
	assert.False(t, hasChanges)
}

// Tests for ParseCommitRange

func TestParseCommitRange_ThreeDotSyntax(t *testing.T) {
	from, to, isRange := ParseCommitRange("abc123...def456")

	assert.Equal(t, "abc123", from)
	assert.Equal(t, "def456", to)
	assert.True(t, isRange)
}

func TestParseCommitRange_TwoDotSyntax(t *testing.T) {
	from, to, isRange := ParseCommitRange("abc123..def456")

	assert.Equal(t, "abc123", from)
	assert.Equal(t, "def456", to)
	assert.True(t, isRange)
}

func TestParseCommitRange_SingleCommit(t *testing.T) {
	from, to, isRange := ParseCommitRange("abc123")

	assert.Equal(t, "", from)
	assert.Equal(t, "abc123", to)
	assert.False(t, isRange)
}

func TestParseCommitRange_HEAD(t *testing.T) {
	from, to, isRange := ParseCommitRange("HEAD")

	assert.Equal(t, "", from)
	assert.Equal(t, "HEAD", to)
	assert.False(t, isRange)
}

func TestParseCommitRange_HEADTilde(t *testing.T) {
	from, to, isRange := ParseCommitRange("HEAD~5...HEAD")

	assert.Equal(t, "HEAD~5", from)
	assert.Equal(t, "HEAD", to)
	assert.True(t, isRange)
}

func TestParseCommitRange_ThreeDotPreferredOverTwoDot(t *testing.T) {
	// Edge case: input contains both ... and ..
	// Should split on ... first
	from, to, isRange := ParseCommitRange("a..b...c")

	assert.Equal(t, "a..b", from)
	assert.Equal(t, "c", to)
	assert.True(t, isRange)
}

// Tests for isAncestor

func TestIsAncestor_True(t *testing.T) {
	tmpDir := t.TempDir()
	setupGitRepo(t, tmpDir)

	// Create first commit
	createFile(t, tmpDir, "first.txt", "content")
	gitAdd(t, tmpDir, "first.txt")
	firstCommit := gitCommitAndGetSHA(t, tmpDir, "First commit")

	// Create second commit
	createFile(t, tmpDir, "second.txt", "content")
	gitAdd(t, tmpDir, "second.txt")
	secondCommit := gitCommitAndGetSHA(t, tmpDir, "Second commit")

	// First should be ancestor of second
	result, err := isAncestor(tmpDir, firstCommit, secondCommit)

	require.NoError(t, err)
	assert.True(t, result)
}

func TestIsAncestor_False(t *testing.T) {
	tmpDir := t.TempDir()
	setupGitRepo(t, tmpDir)

	// Create first commit
	createFile(t, tmpDir, "first.txt", "content")
	gitAdd(t, tmpDir, "first.txt")
	firstCommit := gitCommitAndGetSHA(t, tmpDir, "First commit")

	// Create second commit
	createFile(t, tmpDir, "second.txt", "content")
	gitAdd(t, tmpDir, "second.txt")
	secondCommit := gitCommitAndGetSHA(t, tmpDir, "Second commit")

	// Second should NOT be ancestor of first
	result, err := isAncestor(tmpDir, secondCommit, firstCommit)

	require.NoError(t, err)
	assert.False(t, result)
}

func TestIsAncestor_SameCommit(t *testing.T) {
	tmpDir := t.TempDir()
	setupGitRepo(t, tmpDir)

	createFile(t, tmpDir, "test.txt", "content")
	gitAdd(t, tmpDir, "test.txt")
	commit := gitCommitAndGetSHA(t, tmpDir, "Commit")

	// A commit is its own ancestor
	result, err := isAncestor(tmpDir, commit, commit)

	require.NoError(t, err)
	assert.True(t, result)
}

// Tests for NormalizeCommitRange

func TestNormalizeCommitRange_AlreadyCorrectOrder(t *testing.T) {
	tmpDir := t.TempDir()
	setupGitRepo(t, tmpDir)

	createFile(t, tmpDir, "first.txt", "content")
	gitAdd(t, tmpDir, "first.txt")
	older := gitCommitAndGetSHA(t, tmpDir, "First commit")

	createFile(t, tmpDir, "second.txt", "content")
	gitAdd(t, tmpDir, "second.txt")
	newer := gitCommitAndGetSHA(t, tmpDir, "Second commit")

	from, to, swapped, err := NormalizeCommitRange(tmpDir, older, newer)

	require.NoError(t, err)
	assert.Equal(t, older, from)
	assert.Equal(t, newer, to)
	assert.False(t, swapped)
}

func TestNormalizeCommitRange_ReversedOrder(t *testing.T) {
	tmpDir := t.TempDir()
	setupGitRepo(t, tmpDir)

	createFile(t, tmpDir, "first.txt", "content")
	gitAdd(t, tmpDir, "first.txt")
	older := gitCommitAndGetSHA(t, tmpDir, "First commit")

	createFile(t, tmpDir, "second.txt", "content")
	gitAdd(t, tmpDir, "second.txt")
	newer := gitCommitAndGetSHA(t, tmpDir, "Second commit")

	// Pass in reversed order (newer...older)
	from, to, swapped, err := NormalizeCommitRange(tmpDir, newer, older)

	require.NoError(t, err)
	assert.Equal(t, older, from)
	assert.Equal(t, newer, to)
	assert.True(t, swapped)
}

func TestNormalizeCommitRange_SameCommit(t *testing.T) {
	tmpDir := t.TempDir()
	setupGitRepo(t, tmpDir)

	createFile(t, tmpDir, "test.txt", "content")
	gitAdd(t, tmpDir, "test.txt")
	commit := gitCommitAndGetSHA(t, tmpDir, "Commit")

	from, to, swapped, err := NormalizeCommitRange(tmpDir, commit, commit)

	require.NoError(t, err)
	assert.Equal(t, commit, from)
	assert.Equal(t, commit, to)
	assert.False(t, swapped)
}

// Tests for GetCommitRangeLabel

func TestGetCommitRangeLabel_Success(t *testing.T) {
	tmpDir := t.TempDir()
	setupGitRepo(t, tmpDir)

	createFile(t, tmpDir, "first.txt", "content")
	gitAdd(t, tmpDir, "first.txt")
	firstCommit := gitCommitAndGetSHA(t, tmpDir, "First commit")

	createFile(t, tmpDir, "second.txt", "content")
	gitAdd(t, tmpDir, "second.txt")
	secondCommit := gitCommitAndGetSHA(t, tmpDir, "Second commit")

	label, err := GetCommitRangeLabel(tmpDir, firstCommit, secondCommit)

	require.NoError(t, err)
	assert.Contains(t, label, "...")
	// Should be in format "abc123...def456"
	parts := strings.Split(label, "...")
	assert.Len(t, parts, 2)
	assert.True(t, len(parts[0]) >= 7)
	assert.True(t, len(parts[1]) >= 7)
}

func TestGetCommitRangeLabel_WithHEAD(t *testing.T) {
	tmpDir := t.TempDir()
	setupGitRepo(t, tmpDir)

	createFile(t, tmpDir, "first.txt", "content")
	gitAdd(t, tmpDir, "first.txt")
	firstCommit := gitCommitAndGetSHA(t, tmpDir, "First commit")

	createFile(t, tmpDir, "second.txt", "content")
	gitAdd(t, tmpDir, "second.txt")
	gitCommit(t, tmpDir, "Second commit")

	label, err := GetCommitRangeLabel(tmpDir, firstCommit, "HEAD")

	require.NoError(t, err)
	assert.Contains(t, label, "...")
}

func TestGetCommitRangeLabel_InvalidFromCommit(t *testing.T) {
	tmpDir := t.TempDir()
	setupGitRepo(t, tmpDir)

	createFile(t, tmpDir, "test.txt", "content")
	gitAdd(t, tmpDir, "test.txt")
	commit := gitCommitAndGetSHA(t, tmpDir, "Commit")

	_, err := GetCommitRangeLabel(tmpDir, "invalid-sha", commit)

	assert.Error(t, err)
}

func TestGetCommitRangeLabel_InvalidToCommit(t *testing.T) {
	tmpDir := t.TempDir()
	setupGitRepo(t, tmpDir)

	createFile(t, tmpDir, "test.txt", "content")
	gitAdd(t, tmpDir, "test.txt")
	commit := gitCommitAndGetSHA(t, tmpDir, "Commit")

	_, err := GetCommitRangeLabel(tmpDir, commit, "invalid-sha")

	assert.Error(t, err)
}
