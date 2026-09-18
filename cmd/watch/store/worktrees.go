package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/LegacyCodeHQ/clarity/cmd/watch/protocol"
)

// RegisterWorktree upserts a worktrees row for id. On first sight it's
// inserted with first_seen_at set to now. On a later re-registration (e.g.
// clarity watch relaunched against the same worktree, which is why id is
// deterministic) it refreshes last_known_label and clears disposed_at and
// hidden_at: the supervisor calling this at all means the worktree is
// verifiably live right now, which contradicts any disposed/hidden marker
// left over from a prior lifecycle at this same id. first_seen_at, path,
// and kind are left untouched on conflict — "first" stays first.
func RegisterWorktree(db *sql.DB, id, projectID, path string, kind protocol.WorktreeKind, lastKnownLabel string) error {
	_, err := db.Exec(`
		INSERT INTO worktrees (id, project_id, path, kind, last_known_label, first_seen_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT (id) DO UPDATE SET
			last_known_label = excluded.last_known_label,
			disposed_at = NULL,
			hidden_at = NULL
	`, id, projectID, path, string(kind), lastKnownLabel, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("register worktree %s: %w", id, err)
	}
	return nil
}

// MarkWorktreeDisposed sets disposed_at on worktree id: its underlying git
// worktree was physically removed. The row is never deleted.
func MarkWorktreeDisposed(db *sql.DB, id string) error {
	if _, err := db.Exec(`UPDATE worktrees SET disposed_at = ? WHERE id = ?`, time.Now().UTC(), id); err != nil {
		return fmt.Errorf("mark worktree %s disposed: %w", id, err)
	}
	return nil
}

// MarkWorktreeHidden sets hidden_at on worktree id: the user closed its
// already-disposed tab. The row is never deleted.
func MarkWorktreeHidden(db *sql.DB, id string) error {
	if _, err := db.Exec(`UPDATE worktrees SET hidden_at = ? WHERE id = ?`, time.Now().UTC(), id); err != nil {
		return fmt.Errorf("mark worktree %s hidden: %w", id, err)
	}
	return nil
}
