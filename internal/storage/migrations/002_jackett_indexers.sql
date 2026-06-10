-- internal/storage/migrations/002_jackett_indexers.sql

-- Per-indexer monitoring preferences for Jackett instances.
-- monitored = 1 means Galactica will test this indexer; default is monitored.
CREATE TABLE IF NOT EXISTS jackett_indexer_settings (
    instance_id TEXT NOT NULL REFERENCES instances(id) ON DELETE CASCADE,
    indexer_id  TEXT NOT NULL,
    monitored   INTEGER NOT NULL DEFAULT 1,
    PRIMARY KEY (instance_id, indexer_id)
);
