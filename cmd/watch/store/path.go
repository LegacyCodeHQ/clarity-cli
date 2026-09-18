package store

import (
	"os"
	"path/filepath"

	"github.com/LegacyCodeHQ/clarity/vcs/git"
)

// dbDirName places the session-history database (and the repo lock file)
// inside the repository's common git directory, not a per-worktree path:
// every worktree of the same clone shares one git-common-dir, so this is the
// one location naturally shared across worktrees.id rows for the same
// projects.id. It is also never committed (it lives inside .git) and
// disappears with the repo.
const (
	dbDirName    = "clarity"
	dbFileName   = "session-history.db"
	lockFileName = "watch.lock"
)

// PathFor resolves the session-history database path for the repository
// containing worktreePath. It does not create the database itself, only the
// containing directory, so callers can open/migrate it immediately after.
func PathFor(worktreePath string) (string, error) {
	return resolvePath(worktreePath, dbFileName)
}

// LockPathFor resolves the path to the repo-wide advisory lock file that
// gates a single clarity watch process per repository (see CLR-91). It sits
// beside the session-history database, sharing the same directory and the
// same one-per-clone guarantee.
func LockPathFor(worktreePath string) (string, error) {
	return resolvePath(worktreePath, lockFileName)
}

func resolvePath(worktreePath, fileName string) (string, error) {
	commonDir, err := git.GetCommonDir(worktreePath)
	if err != nil {
		return "", err
	}
	dir := filepath.Join(commonDir, dbDirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, fileName), nil
}
