-- V3.4.3 Approval Workflows: persistent human-in-the-loop decisions.
CREATE TABLE IF NOT EXISTS approval_requests (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    project_id TEXT NOT NULL DEFAULT '',
    environment TEXT NOT NULL DEFAULT 'production',
    requester_id TEXT NOT NULL DEFAULT '',
    action TEXT NOT NULL,
    target_type TEXT NOT NULL DEFAULT '',
    target_id TEXT NOT NULL DEFAULT '',
    payload_sha256 TEXT NOT NULL,
    reason TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'pending',
    expires_at TIMESTAMPTZ,
    approver_id TEXT,
    decision_reason TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    decided_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT approval_requests_environment_check
        CHECK (environment IN ('development', 'staging', 'production')),
    CONSTRAINT approval_requests_status_check
        CHECK (status IN ('pending', 'approved', 'rejected', 'expired', 'cancelled'))
);
CREATE INDEX IF NOT EXISTS idx_approval_requests_tenant_env_status
    ON approval_requests (tenant_id, environment, status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_approval_requests_requester
    ON approval_requests (tenant_id, requester_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_approval_requests_expiry
    ON approval_requests (status, expires_at)
    WHERE status = 'pending';
CREATE UNIQUE INDEX IF NOT EXISTS idx_approval_requests_pending_target
    ON approval_requests (tenant_id, environment, action, target_type, target_id, payload_sha256)
    WHERE status = 'pending';
