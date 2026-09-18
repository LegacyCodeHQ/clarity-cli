package watch

import (
	"database/sql"
	"fmt"

	"github.com/LegacyCodeHQ/clarity/cmd/watch/store"
)

// setupPersistence opens (creating and migrating if needed) the session-
// history database for worktreePath and resolves its projects row, ready to
// pass to broker.enablePersistence. dbPathOverride, when non-empty (the
// --db-path flag), is used verbatim instead of the default location under
// the repo's git-common-dir — needed for development and testing, where
// reading/writing the real repo's own history database on every run would
// get in the way.
func setupPersistence(worktreePath, dbPathOverride string) (db *sql.DB, projectID string, err error) {
	path := dbPathOverride
	if path == "" {
		path, err = store.PathFor(worktreePath)
		if err != nil {
			return nil, "", fmt.Errorf("resolve session-history db path: %w", err)
		}
	}

	db, err = store.Open(path)
	if err != nil {
		return nil, "", fmt.Errorf("open session-history db: %w", err)
	}

	repoOrigin, err := store.ResolveRepoOrigin(worktreePath)
	if err != nil {
		db.Close()
		return nil, "", fmt.Errorf("resolve repo origin: %w", err)
	}

	projectID, err = store.EnsureProject(db, repoOrigin)
	if err != nil {
		db.Close()
		return nil, "", fmt.Errorf("ensure project row: %w", err)
	}

	return db, projectID, nil
}
