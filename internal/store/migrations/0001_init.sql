-- +goose Up
CREATE TABLE IF NOT EXISTS app_sessions (
    name        TEXT PRIMARY KEY,
    jid         TEXT,
    status      TEXT NOT NULL DEFAULT 'STOPPED',
    created_at  INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS webhooks (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    session_name TEXT NOT NULL,
    url          TEXT NOT NULL,
    secret       TEXT NOT NULL DEFAULT '',
    events       TEXT NOT NULL DEFAULT '',
    enabled      INTEGER NOT NULL DEFAULT 1,
    created_at   INTEGER NOT NULL,
    FOREIGN KEY (session_name) REFERENCES app_sessions(name) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_webhooks_session ON webhooks(session_name);

CREATE TABLE IF NOT EXISTS api_keys (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT NOT NULL,
    key_hash    TEXT NOT NULL UNIQUE,
    created_at  INTEGER NOT NULL,
    last_used   INTEGER
);

-- +goose Down
DROP TABLE IF EXISTS api_keys;
DROP TABLE IF EXISTS webhooks;
DROP TABLE IF EXISTS app_sessions;
