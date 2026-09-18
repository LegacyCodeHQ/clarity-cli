package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// SessionSummary is one session's metadata, with no snapshot/commit
// content — see ListSessions. Kept as this package's own type, independent
// of the wire format (mirrored by protocol.PersistedSessionSummary).
type SessionSummary struct {
	ID            int64
	WorktreeID    string
	RunID         int64
	Number        int
	CreatedAt     time.Time
	ClosedAt      *time.Time
	ClosedReason  string
	SnapshotCount int
	CommitCount   int
}

// SnapshotRecord is one recorded snapshot within a session's timeline, as
// returned by GetSessionDetail.
type SnapshotRecord struct {
	Position  int
	Source    string
	Format    string
	Kind      string
	CreatedAt time.Time
}

// SessionDetail is a session's full content: its summary plus every
// snapshot and commit recorded against it, ordered by position.
type SessionDetail struct {
	SessionSummary
	Snapshots []SnapshotRecord
	Commits   []CommitRecord
}

// ListSessions returns every persisted session across the whole project
// identified by projectID — every worktree, every run — ordered by
// worktree then number. No snapshot/commit content, only counts: cheap
// enough to call unconditionally on attach.
//
// Scoped to the project, not a single worktree: a watch_runs row (CLR-97)
// already spans every worktree the process watches, so a worktree-scoped
// listing would force the caller to fetch per-worktree and stitch runs
// back together itself.
func ListSessions(db *sql.DB, projectID string) ([]SessionSummary, error) {
	query := `
		SELECT s.id, s.worktree_id, s.run_id, s.number, s.created_at, s.closed_at, s.closed_reason,
		       (SELECT COUNT(*) FROM snapshots WHERE session_id = s.id),
		       (SELECT COUNT(*) FROM commits WHERE session_id = s.id)
		FROM sessions s
		JOIN worktrees w ON w.id = s.worktree_id
		WHERE w.project_id = ?
		ORDER BY s.worktree_id, s.number`
	rows, err := db.Query(query, projectID)
	if err != nil {
		return nil, fmt.Errorf("list sessions for project %s: %w", projectID, err)
	}
	defer rows.Close()

	var summaries []SessionSummary
	for rows.Next() {
		summary, err := scanSessionSummary(rows)
		if err != nil {
			return nil, fmt.Errorf("list sessions for project %s: %w", projectID, err)
		}
		summaries = append(summaries, summary)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list sessions for project %s: %w", projectID, err)
	}
	return summaries, nil
}

// GetSessionDetail returns sessionID's full content: its summary, every
// snapshot, and every commit, ordered by position. found is false if no
// session with that id exists.
func GetSessionDetail(db *sql.DB, sessionID int64) (detail SessionDetail, found bool, err error) {
	summary, found, err := sessionSummaryByID(db, sessionID)
	if err != nil || !found {
		return SessionDetail{}, found, err
	}

	snapshots, err := sessionSnapshots(db, sessionID)
	if err != nil {
		return SessionDetail{}, false, err
	}
	commits, err := sessionCommits(db, sessionID)
	if err != nil {
		return SessionDetail{}, false, err
	}
	return SessionDetail{SessionSummary: summary, Snapshots: snapshots, Commits: commits}, true, nil
}

func sessionSummaryByID(db *sql.DB, sessionID int64) (SessionSummary, bool, error) {
	query := `
		SELECT id, worktree_id, run_id, number, created_at, closed_at, closed_reason,
		       (SELECT COUNT(*) FROM snapshots WHERE session_id = sessions.id),
		       (SELECT COUNT(*) FROM commits WHERE session_id = sessions.id)
		FROM sessions WHERE id = ?`
	summary, err := scanSessionSummary(db.QueryRow(query, sessionID))
	if errors.Is(err, sql.ErrNoRows) {
		return SessionSummary{}, false, nil
	}
	if err != nil {
		return SessionSummary{}, false, fmt.Errorf("look up session %d: %w", sessionID, err)
	}
	return summary, true, nil
}

// scanSessionSummary scans one row shaped like ListSessions'/
// sessionSummaryByID's query (same column order) into a SessionSummary.
// *sql.Row and *sql.Rows both satisfy this Scan signature, so one scan
// function serves both callers.
func scanSessionSummary(row interface{ Scan(dest ...any) error }) (SessionSummary, error) {
	var s SessionSummary
	var closedAt sql.NullTime
	var closedReason sql.NullString
	err := row.Scan(
		&s.ID, &s.WorktreeID, &s.RunID, &s.Number, &s.CreatedAt, &closedAt, &closedReason,
		&s.SnapshotCount, &s.CommitCount)
	if err != nil {
		return SessionSummary{}, err
	}
	if closedAt.Valid {
		t := closedAt.Time
		s.ClosedAt = &t
	}
	s.ClosedReason = closedReason.String
	return s, nil
}

func sessionSnapshots(db *sql.DB, sessionID int64) ([]SnapshotRecord, error) {
	rows, err := db.Query(
		`SELECT position, source, format, kind, created_at FROM snapshots WHERE session_id = ? ORDER BY position`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("list snapshots for session %d: %w", sessionID, err)
	}
	defer rows.Close()

	var records []SnapshotRecord
	for rows.Next() {
		var r SnapshotRecord
		if err := rows.Scan(&r.Position, &r.Source, &r.Format, &r.Kind, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("list snapshots for session %d: %w", sessionID, err)
		}
		records = append(records, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list snapshots for session %d: %w", sessionID, err)
	}
	return records, nil
}

func sessionCommits(db *sql.DB, sessionID int64) ([]CommitRecord, error) {
	rows, err := db.Query(`SELECT position, hash, subject FROM commits WHERE session_id = ? ORDER BY position`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("list commits for session %d: %w", sessionID, err)
	}
	defer rows.Close()

	var records []CommitRecord
	for rows.Next() {
		var r CommitRecord
		if err := rows.Scan(&r.Position, &r.Hash, &r.Subject); err != nil {
			return nil, fmt.Errorf("list commits for session %d: %w", sessionID, err)
		}
		records = append(records, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list commits for session %d: %w", sessionID, err)
	}
	return records, nil
}
