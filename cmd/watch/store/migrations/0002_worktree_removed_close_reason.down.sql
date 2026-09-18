-- Reverts to the two-value closed_reason constraint. Any row already using
-- 'worktree_removed' would violate the restored CHECK constraint on copy —
-- that's intentional: a lossy downgrade should fail loudly, not silently
-- coerce data into a value that no longer means what it used to.
CREATE TABLE sessions_new (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    worktree_id   TEXT NOT NULL REFERENCES worktrees (id),
    number        INTEGER NOT NULL,
    created_at    DATETIME NOT NULL,
    closed_at     DATETIME,
    closed_reason TEXT CHECK (closed_reason IN ('committed', 'abandoned'))
);

INSERT INTO sessions_new (id, worktree_id, number, created_at, closed_at, closed_reason)
SELECT id, worktree_id, number, created_at, closed_at, closed_reason FROM sessions;

DROP TABLE sessions;
ALTER TABLE sessions_new RENAME TO sessions;

CREATE INDEX idx_sessions_worktree_id ON sessions (worktree_id);
CREATE UNIQUE INDEX idx_sessions_worktree_id_number ON sessions (worktree_id, number);
