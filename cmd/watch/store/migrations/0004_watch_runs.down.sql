-- Reverts run_id off sessions (preserving every other column, in case a
-- test or a future downgrade genuinely has rows to carry back) and drops
-- watch_runs.
CREATE TABLE sessions_new (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    worktree_id   TEXT NOT NULL REFERENCES worktrees (id),
    number        INTEGER NOT NULL,
    created_at    DATETIME NOT NULL,
    closed_at     DATETIME,
    closed_reason TEXT CHECK (closed_reason IN ('committed', 'discarded', 'worktree_removed', 'stale'))
);

INSERT INTO sessions_new (id, worktree_id, number, created_at, closed_at, closed_reason)
SELECT id, worktree_id, number, created_at, closed_at, closed_reason FROM sessions;

DROP TABLE sessions;
ALTER TABLE sessions_new RENAME TO sessions;

CREATE INDEX idx_sessions_worktree_id ON sessions (worktree_id);
CREATE UNIQUE INDEX idx_sessions_worktree_id_number ON sessions (worktree_id, number);

DROP TABLE watch_runs;
