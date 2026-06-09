-- internal/storage/migrations/001_initial.sql

-- instances: monitored apps added at runtime via UI/API
CREATE TABLE IF NOT EXISTS instances (
    id         TEXT PRIMARY KEY NOT NULL,
    kind       TEXT NOT NULL,
    name       TEXT NOT NULL,
    base_url   TEXT NOT NULL,
    enabled    INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- secrets: AES-GCM encrypted credentials keyed by (instance_id, key_name)
CREATE TABLE IF NOT EXISTS secrets (
    instance_id TEXT NOT NULL REFERENCES instances(id) ON DELETE CASCADE,
    key         TEXT NOT NULL,
    value       BLOB NOT NULL,
    PRIMARY KEY (instance_id, key)
);

-- log_entries: normalised log lines from all providers
CREATE TABLE IF NOT EXISTS log_entries (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    instance_id TEXT NOT NULL REFERENCES instances(id) ON DELETE CASCADE,
    ts          DATETIME NOT NULL,
    level       TEXT NOT NULL,
    source_type TEXT NOT NULL,
    message     TEXT NOT NULL,
    raw         TEXT
);
CREATE INDEX IF NOT EXISTS idx_log_entries_lookup
    ON log_entries (instance_id, ts, level);

-- issues: fingerprinted, deduped error groups with impact scoring
CREATE TABLE IF NOT EXISTS issues (
    id           TEXT PRIMARY KEY NOT NULL,
    instance_id  TEXT NOT NULL REFERENCES instances(id) ON DELETE CASCADE,
    fingerprint  TEXT NOT NULL UNIQUE,
    title        TEXT NOT NULL,
    severity     TEXT NOT NULL,
    impact_score REAL NOT NULL DEFAULT 0,
    status       TEXT NOT NULL DEFAULT 'open',
    first_seen   DATETIME NOT NULL,
    last_seen    DATETIME NOT NULL,
    count        INTEGER NOT NULL DEFAULT 1
);
CREATE INDEX IF NOT EXISTS idx_issues_status ON issues (status);
CREATE INDEX IF NOT EXISTS idx_issues_impact  ON issues (impact_score DESC);

-- samples: time-series metrics collected from providers
CREATE TABLE IF NOT EXISTS samples (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    instance_id TEXT NOT NULL REFERENCES instances(id) ON DELETE CASCADE,
    metric      TEXT NOT NULL,
    ts          DATETIME NOT NULL,
    value       REAL NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_samples_lookup
    ON samples (instance_id, metric, ts);

-- notify_channels: notification provider configs (credentials encrypted)
CREATE TABLE IF NOT EXISTS notify_channels (
    id                TEXT PRIMARY KEY NOT NULL,
    name              TEXT NOT NULL,
    provider          TEXT NOT NULL,
    config_encrypted  BLOB,
    enabled           INTEGER NOT NULL DEFAULT 1,
    notify_on_success INTEGER NOT NULL DEFAULT 0,
    notify_on_failure INTEGER NOT NULL DEFAULT 1,
    created_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- backup_targets: destinations for config backups
CREATE TABLE IF NOT EXISTS backup_targets (
    id               TEXT PRIMARY KEY NOT NULL,
    name             TEXT NOT NULL,
    type             TEXT NOT NULL,
    config_encrypted BLOB,
    retention_days   INTEGER NOT NULL DEFAULT 30,
    enabled          INTEGER NOT NULL DEFAULT 1,
    created_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- backups: individual backup run records
CREATE TABLE IF NOT EXISTS backups (
    id          TEXT PRIMARY KEY NOT NULL,
    target_id   TEXT NOT NULL REFERENCES backup_targets(id) ON DELETE CASCADE,
    instance_id TEXT REFERENCES instances(id) ON DELETE SET NULL,
    ts          DATETIME NOT NULL,
    size_bytes  INTEGER NOT NULL DEFAULT 0,
    status      TEXT NOT NULL DEFAULT 'pending',
    location    TEXT,
    error       TEXT,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- sync_jobs: cross-instance config sync definitions (same-kind only)
CREATE TABLE IF NOT EXISTS sync_jobs (
    id                 TEXT PRIMARY KEY NOT NULL,
    source_instance_id TEXT NOT NULL REFERENCES instances(id) ON DELETE CASCADE,
    target_instance_id TEXT NOT NULL REFERENCES instances(id) ON DELETE CASCADE,
    selectors          TEXT NOT NULL DEFAULT '[]',
    schedule           TEXT,
    enabled            INTEGER NOT NULL DEFAULT 1,
    created_at         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- schedules: cron cadences for backup/sync/collection jobs
CREATE TABLE IF NOT EXISTS schedules (
    id         TEXT PRIMARY KEY NOT NULL,
    job_type   TEXT NOT NULL,
    job_id     TEXT NOT NULL,
    cron       TEXT NOT NULL,
    enabled    INTEGER NOT NULL DEFAULT 1,
    last_run   DATETIME,
    next_run   DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- settings: arbitrary key-value application settings
CREATE TABLE IF NOT EXISTS settings (
    key        TEXT PRIMARY KEY NOT NULL,
    value      TEXT NOT NULL,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- api_tokens: bearer tokens for programmatic access
CREATE TABLE IF NOT EXISTS api_tokens (
    id         TEXT PRIMARY KEY NOT NULL,
    name       TEXT NOT NULL,
    token_hash TEXT NOT NULL UNIQUE,
    last_used  DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- admin: single admin user; id=1 enforced by CHECK constraint
CREATE TABLE IF NOT EXISTS admin (
    id            INTEGER PRIMARY KEY CHECK (id = 1),
    username      TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
