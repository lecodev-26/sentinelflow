-- V3.4.2: persist the authenticated API-key environment through the data/event planes.
-- Existing records are production for backwards compatibility.
ALTER TABLE usage_records
    ADD COLUMN IF NOT EXISTS environment TEXT NOT NULL DEFAULT 'production';
ALTER TABLE usage_records
    DROP CONSTRAINT IF EXISTS usage_records_environment_check;
ALTER TABLE usage_records
    ADD CONSTRAINT usage_records_environment_check
    CHECK (environment IN ('development', 'staging', 'production'));
CREATE INDEX IF NOT EXISTS idx_usage_tenant_environment_time
    ON usage_records (tenant_id, environment, timestamp DESC);

ALTER TABLE analytics_daily
    ADD COLUMN IF NOT EXISTS environment TEXT NOT NULL DEFAULT 'production';
ALTER TABLE analytics_daily DROP CONSTRAINT IF EXISTS analytics_daily_pkey;
ALTER TABLE analytics_daily ADD PRIMARY KEY (tenant_id, environment, day);
CREATE INDEX IF NOT EXISTS idx_analytics_environment_day
    ON analytics_daily (tenant_id, environment, day DESC);

ALTER TABLE audit_log
    ADD COLUMN IF NOT EXISTS environment TEXT NOT NULL DEFAULT 'production';
CREATE INDEX IF NOT EXISTS idx_audit_tenant_environment_time
    ON audit_log (tenant_id, environment, occurred_at DESC);

ALTER TABLE security_events
    ADD COLUMN IF NOT EXISTS environment TEXT NOT NULL DEFAULT 'production';
CREATE INDEX IF NOT EXISTS idx_security_tenant_environment_time
    ON security_events (tenant_id, environment, occurred_at DESC);
