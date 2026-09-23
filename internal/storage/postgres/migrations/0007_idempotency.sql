-- Distributed idempotency store
--
-- Replaces in-memory map with PostgreSQL-backed store.
-- Allows multiple gateway instances to share idempotency state.

CREATE TABLE IF NOT EXISTS idempotency_keys (
    key          TEXT PRIMARY KEY,
    tenant_id    TEXT NOT NULL,
    request_hash TEXT NOT NULL,  -- SHA-256 of request body to detect key reuse
    status       TEXT NOT NULL,   -- 'pending' | 'completed' | 'failed'
    response     BYTEA,           -- response body (completed only)
    status_code  INT,             -- HTTP status code (completed only)
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at   TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_idempotency_expires
    ON idempotency_keys(expires_at);

CREATE INDEX IF NOT EXISTS idx_idempotency_tenant
    ON idempotency_keys(tenant_id);

-- Auto-cleanup function (called periodically)
CREATE OR REPLACE FUNCTION cleanup_expired_idempotency()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM idempotency_keys
    WHERE expires_at < NOW();
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Enable RLS (security)
ALTER TABLE idempotency_keys ENABLE ROW LEVEL SECURITY;
