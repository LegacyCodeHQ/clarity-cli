-- Renames sessions.closed_reason 'abandoned' -> 'discarded' (more precise:
-- names what happened to the changes, not a vague characterization) and
-- adds 'stale': a session left open by a crashed/killed process, found at
-- the next restart to no longer match the current state, and closed rather
-- than resumed. See migration 0002 for why this needs a table rebuild and
-- why Migrate() runs it with foreign key enforcement off.
--
-- The rename has to happen as part of the copy below (the CASE), not as a
-- separate UPDATE against the old table first: the old table's own CHECK
-- constraint doesn't allow 'discarded' either, so renaming in place before
-- the rebuild fails immediately — confirmed by reproducing it directly
-- against a throwaway sqlite3 database before writing this migration.
CREATE TABLE sessions_new (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    worktree_id   TEXT NOT NULL REFERENCES worktrees (id),
    number        INTEGER NOT NULL,
    created_at    DATETIME NOT NULL,
    closed_at     DATETIME,
    closed_reason TEXT CHECK (closed_reason IN ('committed', 'discarded', 'worktree_removed', 'stale'))
);

INSERT INTO sessions_new (id, worktree_id, number, created_at, closed_at, closed_reason)
SELECT id, worktree_id, number, created_at, closed_at,
       CASE closed_reason WHEN 'abandoned' THEN 'discarded' ELSE closed_reason END
FROM sessions;

DROP TABLE sessions;
ALTER TABLE sessions_new RENAME TO sessions;

CREATE INDEX idx_sessions_worktree_id ON sessions (worktree_id);
CREATE UNIQUE INDEX idx_sessions_worktree_id_number ON sessions (worktree_id, number);
