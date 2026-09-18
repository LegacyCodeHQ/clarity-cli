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
// prior process run (resume vs. CloseSessionStale) is restart-hydration's
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
// design; a session with none belongs to CloseSessionDiscarded,
// CloseSessionWorktreeRemoved, or CloseSessionStale instead.
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

// CloseSessionDiscarded closes sessionID as discarded: the working tree
// returned to a clean state with no commit (stash, checkout -- ., reset
// with no reachable new commits) while clarity watch was actively watching.
// No commits are recorded.
func CloseSessionDiscarded(db *sql.DB, sessionID int64) error {
	return closeSessionWithoutCommit(db, sessionID, "discarded")
}

// CloseSessionWorktreeRemoved closes sessionID as worktree_removed: the
// underlying git worktree was physically removed while the session still
// had uncommitted snapshots. No commits are recorded.
func CloseSessionWorktreeRemoved(db *sql.DB, sessionID int64) error {
	return closeSessionWithoutCommit(db, sessionID, "worktree_removed")
}

// CloseSessionStale closes sessionID as stale: it was left open by a
// process that exited without closing it (a crash, a kill, a machine
// restart — clarity watch doesn't know which), and at the next relaunch its
// last recorded snapshot no longer matched the freshly rebuilt graph, so it
// was closed rather than resumed. No commits are recorded. Called by
// OpenOrResumeSession — not normally called directly.
func CloseSessionStale(db *sql.DB, sessionID int64) error {
	return closeSessionWithoutCommit(db, sessionID, "stale")
}

func closeSessionWithoutCommit(db *sql.DB, sessionID int64, reason string) error {
	closeSQL := `UPDATE sessions SET closed_at = ?, closed_reason = ? WHERE id = ?`
	if _, err := db.Exec(closeSQL, time.Now().UTC(), reason, sessionID); err != nil {
		return fmt.Errorf("close session %d as %s: %w", sessionID, reason, err)
	}
	return nil
}

// OpenOrResumeSession is the entry point for the first snapshot of a
// worktree's lifetime in this process — the restart-hydration decision.
//
// If no session was left open for worktreeID, this is exactly OpenSession:
// a fresh session opens, matched is false, nextPosition is 0.
//
// If a session WAS left open (a prior process run never got to close it —
// crash, kill, machine restart), it's a resume candidate. currentSource is
// the freshly rebuilt graph for this worktree, always available for free
// since attaching to a worktree already rebuilds it to seed the tab. If it
// matches that session's last recorded snapshot exactly, the gap was
// invisible to what we actually track, so the session is resumed: matched
// is true, telling the caller currentSource is already recorded and must
// not be written again, and nextPosition is where the next *new* snapshot
// should go. If it doesn't match, the orphaned session is closed via
// CloseSessionStale (never resumed, never silently left open) and a fresh
// session opens normally.
func OpenOrResumeSession(db *sql.DB, worktreeID, currentSource string) (sessionID int64, nextPosition int, matched bool, err error) {
	openID, found, err := latestOpenSession(db, worktreeID)
	if err != nil {
		return 0, 0, false, err
	}
	if found {
		lastSource, lastPosition, hasSnapshot, err := lastSnapshot(db, openID)
		if err != nil {
			return 0, 0, false, err
		}
		if hasSnapshot && lastSource == currentSource {
			return openID, lastPosition + 1, true, nil
		}
		if err := CloseSessionStale(db, openID); err != nil {
			return 0, 0, false, fmt.Errorf("close stale session for worktree %s: %w", worktreeID, err)
		}
	}

	newID, err := OpenSession(db, worktreeID)
	if err != nil {
		return 0, 0, false, err
	}
	return newID, 0, false, nil
}

// latestOpenSession returns the highest-numbered still-open session
// (closed_at IS NULL) for worktreeID, if any. Only the highest is ever a
// resume candidate — an older open session left behind is a pre-existing
// anomaly from before this logic existed, not this restart's problem.
func latestOpenSession(db *sql.DB, worktreeID string) (sessionID int64, found bool, err error) {
	querySQL := `SELECT id FROM sessions WHERE worktree_id = ? AND closed_at IS NULL ORDER BY number DESC LIMIT 1`
	err = db.QueryRow(querySQL, worktreeID).Scan(&sessionID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("look up open session for worktree %s: %w", worktreeID, err)
	}
	return sessionID, true, nil
}

// lastSnapshot returns the most recently recorded snapshot for sessionID,
// if it has any. A session can exist with zero snapshots if a process
// crashed between OpenSession succeeding and the first AppendSnapshot —
// found is false in that case, since there's nothing to compare against.
func lastSnapshot(db *sql.DB, sessionID int64) (source string, position int, found bool, err error) {
	querySQL := `SELECT source, position FROM snapshots WHERE session_id = ? ORDER BY position DESC LIMIT 1`
	err = db.QueryRow(querySQL, sessionID).Scan(&source, &position)
	if errors.Is(err, sql.ErrNoRows) {
		return "", 0, false, nil
	}
	if err != nil {
		return "", 0, false, fmt.Errorf("look up last snapshot for session %d: %w", sessionID, err)
	}
	return source, position, true, nil
}
