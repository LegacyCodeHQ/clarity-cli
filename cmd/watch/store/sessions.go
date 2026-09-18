package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/mattn/go-sqlite3"
)

const maxOpenSessionAttempts = 5

// CommitRecord is one commit to record against a session being closed as
// committed.
type CommitRecord struct {
	Position int
	Hash     string
	Subject  string
}

// OpenSession creates a new open session (closed_at NULL) for worktreeID,
// numbered one past the highest number ever used for that worktree —
// closed or still open, so a stale open session left behind by an earlier
// process run is never renumbered or reused — and returns its id.
//
// Each call always creates a fresh session; it never looks for or resumes
// a previously-open one. Deciding what to do with a session left open by a
// prior process run (resume vs. treat as abandoned) is restart-hydration's
// job, not this one's.
//
// Numbering is computed as MAX(number)+1 inside a transaction and retried
// on a unique-constraint conflict against idx_sessions_worktree_id_number,
// so it stays correct under concurrent writers regardless of what CLR-91
// concludes about process-level coordination.
func OpenSession(db *sql.DB, worktreeID string) (int64, error) {
	for attempt := 0; attempt < maxOpenSessionAttempts; attempt++ {
		id, err := tryOpenSession(db, worktreeID)
		if err == nil {
			return id, nil
		}
		if !isUniqueConstraintErr(err) {
			return 0, fmt.Errorf("open session for worktree %s: %w", worktreeID, err)
		}
		// Lost a race for this number against another writer; retry with a
		// fresh MAX read.
	}
	return 0, fmt.Errorf("open session for worktree %s: too many conflicting concurrent writers", worktreeID)
}

func tryOpenSession(db *sql.DB, worktreeID string) (int64, error) {
	var maxNumber sql.NullInt64
	if err := db.QueryRow(`SELECT MAX(number) FROM sessions WHERE worktree_id = ?`, worktreeID).Scan(&maxNumber); err != nil {
		return 0, fmt.Errorf("compute next session number: %w", err)
	}
	number := maxNumber.Int64 + 1 // NULL (no rows yet) scans as 0 -> first number is 1.

	insertSQL := `INSERT INTO sessions (worktree_id, number, created_at) VALUES (?, ?, ?)`
	res, err := db.Exec(insertSQL, worktreeID, number, time.Now().UTC())
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func isUniqueConstraintErr(err error) bool {
	var sqliteErr sqlite3.Error
	return errors.As(err, &sqliteErr) && sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique
}

// CloseSessionCommitted closes sessionID as committed: sets closed_at and
// closed_reason='committed', and records commits. Requires at least one
// commit — a session closed by a real commit always has one, per the
// design; a session with none belongs to CloseSessionAbandoned or
// CloseSessionWorktreeRemoved instead.
func CloseSessionCommitted(db *sql.DB, sessionID int64, commits []CommitRecord) error {
	if len(commits) == 0 {
		return fmt.Errorf("close session %d as committed: at least one commit required", sessionID)
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("close session %d as committed: %w", sessionID, err)
	}
	defer tx.Rollback() //nolint:errcheck // no-op once Commit succeeds

	closeSQL := `UPDATE sessions SET closed_at = ?, closed_reason = 'committed' WHERE id = ?`
	if _, err := tx.Exec(closeSQL, time.Now().UTC(), sessionID); err != nil {
		return fmt.Errorf("close session %d as committed: %w", sessionID, err)
	}
	insertCommitSQL := `INSERT INTO commits (session_id, position, hash, subject) VALUES (?, ?, ?, ?)`
	for _, c := range commits {
		if _, err := tx.Exec(insertCommitSQL, sessionID, c.Position, c.Hash, c.Subject); err != nil {
			return fmt.Errorf("close session %d as committed: insert commit: %w", sessionID, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("close session %d as committed: %w", sessionID, err)
	}
	return nil
}

// CloseSessionAbandoned closes sessionID as abandoned: the working tree
// returned to a clean state with no commit (stash, checkout -- ., reset
// with no reachable new commits). No commits are recorded.
func CloseSessionAbandoned(db *sql.DB, sessionID int64) error {
	return closeSessionWithoutCommit(db, sessionID, "abandoned")
}

// CloseSessionWorktreeRemoved closes sessionID as worktree_removed: the
// underlying git worktree was physically removed while the session still
// had uncommitted snapshots. No commits are recorded.
func CloseSessionWorktreeRemoved(db *sql.DB, sessionID int64) error {
	return closeSessionWithoutCommit(db, sessionID, "worktree_removed")
}

func closeSessionWithoutCommit(db *sql.DB, sessionID int64, reason string) error {
	closeSQL := `UPDATE sessions SET closed_at = ?, closed_reason = ? WHERE id = ?`
	if _, err := db.Exec(closeSQL, time.Now().UTC(), reason, sessionID); err != nil {
		return fmt.Errorf("close session %d as %s: %w", sessionID, reason, err)
	}
	return nil
}
