// Package store persists clarity watch's session history (projects,
// worktrees, sessions, commits, snapshots) to a per-repo SQLite database, so
// it survives a process restart or crash. See the "ER: Persisted Session
// History" and "Session Lifecycle" design artifacts (Strata, Clarity
// project) for the schema and its rationale.
package store

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

// Open opens the SQLite database at path with the pragmas this package
// relies on (foreign key enforcement, since the schema depends on it) and
// runs any pending migrations. Callers should PathFor(worktreePath) to get
// path.
func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", path+"?_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	if err := Migrate(db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}
