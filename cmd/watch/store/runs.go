package store

import (
	"database/sql"
	"fmt"
	"time"
)

// OpenRun creates a new watch_runs row for the clarity watch process
// identified by pid, watching the repo identified by projectID, and
// returns its id.
//
// A run's boundary is deliberately the same one CLR-91's repo lock already
// enforces: one process holds the lock for its whole lifetime, watching the
// entire repo (main worktree plus every linked worktree) under that single
// hold. OpenRun should be called once per process, right after that lock is
// acquired and persistence is otherwise set up — every session opened
// afterward is stamped with this run's id (see OpenSession), so which
// `clarity watch` invocation produced a given session becomes recoverable
// without inferring it from timestamps.
func OpenRun(db *sql.DB, projectID string, pid int) (int64, error) {
	insertSQL := `INSERT INTO watch_runs (project_id, pid, started_at) VALUES (?, ?, ?)`
	res, err := db.Exec(insertSQL, projectID, pid, time.Now().UTC())
	if err != nil {
		return 0, fmt.Errorf("open watch run for project %s: %w", projectID, err)
	}
	return res.LastInsertId()
}

// CloseRun sets ended_at on runID, marking it as having shut down cleanly.
// A run whose ended_at is still NULL crashed or was killed rather than
// exiting through its own defer chain — the same "NULL means unclean exit"
// convention CloseSessionStale relies on for a session left open.
func CloseRun(db *sql.DB, runID int64) error {
	closeSQL := `UPDATE watch_runs SET ended_at = ? WHERE id = ?`
	if _, err := db.Exec(closeSQL, time.Now().UTC(), runID); err != nil {
		return fmt.Errorf("close watch run %d: %w", runID, err)
	}
	return nil
}
