-- Adds 'worktree_removed' to sessions.closed_reason: a session can also
-- close because its worktree was physically removed while snapshots were
-- still accumulating (markWorktreeFinished archives with no commits),
-- distinct from the user reverting to a clean tree without committing
-- ('abandoned'). SQLite has no ALTER TABLE for CHECK constraints, so the
-- table is rebuilt.
--
-- Safe to copy every row unconditionally here (see migrate.go for the
-- foreign-key-off handling this relies on): sessions rows are never
-- deleted by this application, only marked closed, so there is no gap
-- between the copied rows' max id and the table's true historical max —
-- a rebuild that copies fewer rows than were ever inserted (e.g. after a
-- DELETE) would need to reseed sqlite_sequence explicitly to avoid
-- reusing a since-freed id.
CREATE TABLE sessions_new (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    worktree_id   TEXT NOT NULL REFERENCES worktrees (id),
    number        INTEGER NOT NULL,
    created_at    DATETIME NOT NULL,
    closed_at     DATETIME,
    closed_reason TEXT CHECK (closed_reason IN ('committed', 'abandoned', 'worktree_removed'))
);

INSERT INTO sessions_new (id, worktree_id, number, created_at, closed_at, closed_reason)
SELECT id, worktree_id, number, created_at, closed_at, closed_reason FROM sessions;

DROP TABLE sessions;
ALTER TABLE sessions_new RENAME TO sessions;

CREATE INDEX idx_sessions_worktree_id ON sessions (worktree_id);
CREATE UNIQUE INDEX idx_sessions_worktree_id_number ON sessions (worktree_id, number);
