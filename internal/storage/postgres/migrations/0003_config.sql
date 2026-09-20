-- Config versions
CREATE TABLE IF NOT EXISTS config_versions (
    version INTEGER PRIMARY KEY,
    config JSONB NOT NULL,
    applied_by TEXT,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    active BOOLEAN NOT NULL DEFAULT FALSE
);
