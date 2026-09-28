-- V3.4.2 Environments: bind each API key to an explicit runtime environment.
-- Existing keys are production by default for backwards compatibility.
ALTER TABLE api_keys
    ADD COLUMN IF NOT EXISTS environment TEXT NOT NULL DEFAULT 'production';

ALTER TABLE api_keys
    DROP CONSTRAINT IF EXISTS api_keys_environment_check;

ALTER TABLE api_keys
    ADD CONSTRAINT api_keys_environment_check
    CHECK (environment IN ('development', 'staging', 'production'));

CREATE INDEX IF NOT EXISTS idx_apikeys_org_environment
    ON api_keys(org_id, environment);

CREATE INDEX IF NOT EXISTS idx_apikeys_project_environment
    ON api_keys(project_id, environment);
