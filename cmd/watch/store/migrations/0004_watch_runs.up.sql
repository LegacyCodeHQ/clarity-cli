-- Adds watch_runs, the boundary the CLR-91 repo lock's hold already is:
-- one process acquires the lock once at startup and releases it once at
-- exit, watching the whole repo (main worktree + every linked worktree)
-- under that single hold. A row here records one such hold; sessions
-- opened during it are stamped with its id via the new sessions.run_id,
-- so "which watch invocation produced these sessions" becomes answerable
-- without inferring it from timestamps. ended_at stays NULL for a run
-- that crashed rather than shut down cleanly, same spirit as an orphaned
-- session's own closed_at.
CREATE TABLE watch_runs (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id TEXT NOT NULL REFERENCES projects (id),
    pid        INTEGER NOT NULL,
    started_at DATETIME NOT NULL,
    ended_at   DATETIME
);

CREATE INDEX idx_watch_runs_project_id ON watch_runs (project_id);

-- sessions.run_id is NOT NULL with no default -- unlike migration 0002/0003,
-- which each rebuilt sessions to preserve real pre-existing rows (this
-- schema had shipped and could already hold data by then), this feature has
-- never shipped in a release: v0.32.0 was tagged before CLR-93 introduced
-- this table, and there is no session-history.db anywhere with rows against
-- this schema to preserve. So there's nothing to backfill and no reason to
-- make run_id nullable to absorb migration debt that doesn't exist -- every
-- session from this point on is only ever created by OpenSession, which
-- only ever runs from inside an already-persisting `clarity watch` process,
-- meaning a run row always exists by construction.
DROP TABLE sessions;

CREATE TABLE sessions (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    worktree_id   TEXT NOT NULL REFERENCES worktrees (id),
    run_id        INTEGER NOT NULL REFERENCES watch_runs (id),
    number        INTEGER NOT NULL,
    created_at    DATETIME NOT NULL,
    closed_at     DATETIME,
    closed_reason TEXT CHECK (closed_reason IN ('committed', 'discarded', 'worktree_removed', 'stale'))
);

CREATE INDEX idx_sessions_worktree_id ON sessions (worktree_id);
CREATE UNIQUE INDEX idx_sessions_worktree_id_number ON sessions (worktree_id, number);
CREATE INDEX idx_sessions_run_id ON sessions (run_id);
