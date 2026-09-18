-- Reverts to the three-value constraint from migration 0002. The rename
-- back happens inline in the copy below, same reasoning as the up
-- migration. Any row using 'stale' is left untouched by the CASE and so
-- still fails the restored CHECK constraint on copy — intentional, same
-- reasoning as 0002's down migration: a lossy downgrade should fail
-- loudly, not silently coerce data into a value that no longer applies.
CREATE TABLE sessions_new (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    worktree_id   TEXT NOT NULL REFERENCES worktrees (id),
    number        INTEGER NOT NULL,
    created_at    DATETIME NOT NULL,
    closed_at     DATETIME,
    closed_reason TEXT CHECK (closed_reason IN ('committed', 'abandoned', 'worktree_removed'))
);

INSERT INTO sessions_new (id, worktree_id, number, created_at, closed_at, closed_reason)
SELECT id, worktree_id, number, created_at, closed_at,
       CASE closed_reason WHEN 'discarded' THEN 'abandoned' ELSE closed_reason END
FROM sessions;

DROP TABLE sessions;
ALTER TABLE sessions_new RENAME TO sessions;

CREATE INDEX idx_sessions_worktree_id ON sessions (worktree_id);
CREATE UNIQUE INDEX idx_sessions_worktree_id_number ON sessions (worktree_id, number);
