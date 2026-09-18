package store

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/LegacyCodeHQ/clarity/vcs/git"
)

// ResolveRepoOrigin returns the stable identity this package uses for
// projects.repo_origin: the repo's "origin" remote URL, or — for a repo
// with no remote configured, the normal state for a fresh `git init` that
// hasn't been pushed anywhere — a synthetic identity derived from its
// common git dir, prefixed so it can never collide with a real remote URL.
func ResolveRepoOrigin(worktreePath string) (string, error) {
	url, err := git.RemoteOriginURL(worktreePath)
	if err != nil {
		return "", err
	}
	if url != "" {
		return url, nil
	}
	commonDir, err := git.GetCommonDir(worktreePath)
	if err != nil {
		return "", err
	}
	return "local:" + commonDir, nil
}

// EnsureProject returns the id of the projects row for repoOrigin, creating
// one (with a fresh UUID) if it doesn't exist yet. Idempotent: calling it
// again for the same repoOrigin always returns the same id, never inserts a
// duplicate row — including under a race, since repo_origin is UNIQUE and a
// conflicting insert falls back to re-reading the winning row.
func EnsureProject(db *sql.DB, repoOrigin string) (string, error) {
	id, err := lookupProjectID(db, repoOrigin)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("look up project for %s: %w", repoOrigin, err)
	}

	id = uuid.NewString()
	insertSQL := `INSERT INTO projects (id, repo_origin, created_at) VALUES (?, ?, datetime('now'))`
	_, insertErr := db.Exec(insertSQL, id, repoOrigin)
	if insertErr == nil {
		return id, nil
	}

	// Someone else created it between our lookup and this insert.
	if id, err := lookupProjectID(db, repoOrigin); err == nil {
		return id, nil
	}
	return "", fmt.Errorf("create project for %s: %w", repoOrigin, insertErr)
}

func lookupProjectID(db *sql.DB, repoOrigin string) (string, error) {
	var id string
	err := db.QueryRow(`SELECT id FROM projects WHERE repo_origin = ?`, repoOrigin).Scan(&id)
	return id, err
}
