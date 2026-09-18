CREATE TABLE projects (
    id          TEXT PRIMARY KEY,
    repo_origin TEXT NOT NULL UNIQUE,
    created_at  DATETIME NOT NULL
);

CREATE TABLE worktrees (
    id                TEXT PRIMARY KEY,
    project_id        TEXT NOT NULL REFERENCES projects (id),
    path              TEXT NOT NULL,
    kind              TEXT NOT NULL CHECK (kind IN ('main', 'linked')),
    last_known_label  TEXT,
    first_seen_at     DATETIME NOT NULL,
    disposed_at       DATETIME,
    hidden_at         DATETIME
);

CREATE INDEX idx_worktrees_project_id ON worktrees (project_id);

CREATE TABLE sessions (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    worktree_id   TEXT NOT NULL REFERENCES worktrees (id),
    number        INTEGER NOT NULL,
    created_at    DATETIME NOT NULL,
    closed_at     DATETIME,
    closed_reason TEXT CHECK (closed_reason IN ('committed', 'abandoned'))
);

CREATE INDEX idx_sessions_worktree_id ON sessions (worktree_id);
CREATE UNIQUE INDEX idx_sessions_worktree_id_number ON sessions (worktree_id, number);

CREATE TABLE commits (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id INTEGER NOT NULL REFERENCES sessions (id),
    position   INTEGER NOT NULL,
    hash       TEXT NOT NULL,
    subject    TEXT NOT NULL
);

CREATE INDEX idx_commits_session_id ON commits (session_id);

CREATE TABLE snapshots (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id INTEGER NOT NULL REFERENCES sessions (id),
    position   INTEGER NOT NULL,
    source     TEXT NOT NULL,
    format     TEXT NOT NULL CHECK (format IN ('dot', 'mermaid')),
    kind       TEXT NOT NULL CHECK (kind IN ('baseline', 'incremental')),
    created_at DATETIME NOT NULL
);

CREATE INDEX idx_snapshots_session_id ON snapshots (session_id);
