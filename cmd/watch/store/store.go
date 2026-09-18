// Package store persists clarity watch's session history (projects,
// worktrees, sessions, commits, snapshots) to a per-repo SQLite database, so
// it survives a process restart or crash. See the "ER: Persisted Session
// History" and "Session Lifecycle" design artifacts (Strata, Clarity
// project) for the schema and its rationale.
package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

// Open opens the SQLite database at path with the pragmas this package
// relies on (foreign key enforcement, since the schema depends on it) and
// runs any pending migrations. Callers should PathFor(worktreePath) to get
// path.
//
// The connection pool is limited to one connection: SQLite serializes
// writers anyway, and Migrate needs a guarantee that the connection it
// toggles PRAGMA foreign_keys on is the same one golang-migrate later runs
// its transaction on (see migrate.go).
func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", path+"?_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		db.Close()
		// go-sqlite3's cgo driver reports "no such file or directory" for a
		// directory permission failure too (confirmed by hand: the same path
		// opened via Go's own os package correctly reports "permission
		// denied"). Re-probe with os directly so a permission problem doesn't
		// get misreported as a missing path.
		if probeErr := probeDirWritable(path); probeErr != nil {
			return nil, fmt.Errorf("open %s: %w", path, probeErr)
		}
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	if err := Migrate(db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

// probeDirWritable creates and immediately removes a throwaway file next to
// path, purely to get Go's own errno when the sqlite driver's error about
// opening path is misleading. Returns nil (no accurate replacement found) if
// the directory is genuinely writable.
func probeDirWritable(path string) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".clarity-writetest-*")
	if err != nil {
		return err
	}
	name := f.Name()
	f.Close()
	os.Remove(name)
	return nil
}
