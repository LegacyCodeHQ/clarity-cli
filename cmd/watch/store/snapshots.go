package store

import (
	"database/sql"
	"fmt"
	"time"
)

// AppendSnapshot writes one snapshot into sessionID's snapshot stream,
// called as each snapshot is produced (not batched) so a crash or kill
// loses at most the single write in flight, never the whole session.
func AppendSnapshot(db *sql.DB, sessionID int64, position int, source, format, kind string, createdAt time.Time) error {
	insertSQL := `INSERT INTO snapshots (session_id, position, source, format, kind, created_at) VALUES (?, ?, ?, ?, ?, ?)`
	_, err := db.Exec(insertSQL, sessionID, position, source, format, kind, createdAt)
	if err != nil {
		return fmt.Errorf("append snapshot for session %d: %w", sessionID, err)
	}
	return nil
}
