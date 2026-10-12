-- V6 durable execution state. Tenant identity is part of every relationship
-- and uniqueness boundary; payloads remain external references.
CREATE TABLE IF NOT EXISTS v6_runs (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    principal_id TEXT NOT NULL,
    idempotency_key TEXT,
    correlation_id TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('pending','queued','running','succeeded','failed','cancelled')),
    payload_ref TEXT NOT NULL DEFAULT '',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    error_code TEXT NOT NULL DEFAULT '',
    UNIQUE (tenant_id, id),
    UNIQUE (tenant_id, idempotency_key),
    CHECK (updated_at >= created_at),
    CHECK (finished_at IS NULL OR (started_at IS NOT NULL AND finished_at >= started_at))
);
CREATE INDEX IF NOT EXISTS idx_v6_runs_tenant_created ON v6_runs (tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_v6_runs_status_updated ON v6_runs (status, updated_at);

CREATE TABLE IF NOT EXISTS v6_tasks (
    id TEXT PRIMARY KEY,
    run_id TEXT NOT NULL,
    tenant_id TEXT NOT NULL,
    correlation_id TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('pending','queued','running','succeeded','failed','cancelled','retrying')),
    attempt INTEGER NOT NULL DEFAULT 0 CHECK (attempt >= 0),
    max_attempts INTEGER NOT NULL CHECK (max_attempts > 0 AND attempt <= max_attempts),
    available_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    lease_owner TEXT NOT NULL DEFAULT '',
    lease_until TIMESTAMPTZ,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    error_code TEXT NOT NULL DEFAULT '',
    UNIQUE (tenant_id, id),
    FOREIGN KEY (tenant_id, run_id) REFERENCES v6_runs (tenant_id, id) ON DELETE CASCADE,
    CHECK (updated_at >= created_at),
    CHECK (lease_until IS NULL OR lease_owner <> ''),
    CHECK (finished_at IS NULL OR (started_at IS NOT NULL AND finished_at >= started_at))
);
CREATE INDEX IF NOT EXISTS idx_v6_tasks_claimable ON v6_tasks (available_at, created_at) WHERE status IN ('queued','retrying');
CREATE INDEX IF NOT EXISTS idx_v6_tasks_expired_lease ON v6_tasks (lease_until) WHERE status = 'running' AND lease_until IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_v6_tasks_run ON v6_tasks (tenant_id, run_id, created_at);

CREATE TABLE IF NOT EXISTS v6_artifacts (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    run_id TEXT NOT NULL,
    task_id TEXT,
    kind TEXT NOT NULL,
    media_type TEXT NOT NULL DEFAULT '',
    storage_ref TEXT NOT NULL,
    checksum TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by TEXT NOT NULL,
    policy_version TEXT NOT NULL DEFAULT '',
    FOREIGN KEY (tenant_id, run_id) REFERENCES v6_runs (tenant_id, id) ON DELETE CASCADE,
    FOREIGN KEY (tenant_id, task_id) REFERENCES v6_tasks (tenant_id, id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_v6_artifacts_run ON v6_artifacts (tenant_id, run_id, created_at);

CREATE TABLE IF NOT EXISTS v6_execution_events (
    id TEXT PRIMARY KEY,
    schema_version INTEGER NOT NULL CHECK (schema_version > 0),
    tenant_id TEXT NOT NULL,
    run_id TEXT NOT NULL,
    task_id TEXT,
    correlation_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL,
    principal_id TEXT NOT NULL DEFAULT '',
    policy_version TEXT NOT NULL DEFAULT '',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    FOREIGN KEY (tenant_id, run_id) REFERENCES v6_runs (tenant_id, id) ON DELETE CASCADE,
    FOREIGN KEY (tenant_id, task_id) REFERENCES v6_tasks (tenant_id, id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_v6_events_run_time ON v6_execution_events (tenant_id, run_id, occurred_at, id);
CREATE INDEX IF NOT EXISTS idx_v6_events_correlation ON v6_execution_events (tenant_id, correlation_id, occurred_at);

ALTER TABLE v6_runs ENABLE ROW LEVEL SECURITY;
ALTER TABLE v6_tasks ENABLE ROW LEVEL SECURITY;
ALTER TABLE v6_artifacts ENABLE ROW LEVEL SECURITY;
ALTER TABLE v6_execution_events ENABLE ROW LEVEL SECURITY;
