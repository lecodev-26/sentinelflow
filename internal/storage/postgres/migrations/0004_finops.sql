-- FinOps: usage records y budgets

CREATE TABLE IF NOT EXISTS usage_records (
    id TEXT PRIMARY KEY,
    request_id TEXT NOT NULL,
    tenant_id TEXT NOT NULL,
    project_id TEXT,
    user_id TEXT,
    api_key_id TEXT,
    provider TEXT NOT NULL,
    model TEXT NOT NULL,
    input_tokens INTEGER NOT NULL DEFAULT 0,
    output_tokens INTEGER NOT NULL DEFAULT 0,
    total_tokens INTEGER NOT NULL DEFAULT 0,
    input_cost_usd REAL NOT NULL DEFAULT 0,
    output_cost_usd REAL NOT NULL DEFAULT 0,
    cost_usd REAL NOT NULL DEFAULT 0,
    latency_ms INTEGER NOT NULL DEFAULT 0,
    ttft_ms INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL,
    cache_hit BOOLEAN NOT NULL DEFAULT FALSE,
    fallback BOOLEAN NOT NULL DEFAULT FALSE,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_usage_tenant ON usage_records(tenant_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_usage_request ON usage_records(request_id);
CREATE INDEX IF NOT EXISTS idx_usage_provider ON usage_records(provider, timestamp DESC);

CREATE TABLE IF NOT EXISTS budgets (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    scope TEXT NOT NULL DEFAULT 'tenant', -- tenant, project, model
    scope_id TEXT,                        -- project_id or model_id (null = tenant-wide)
    monthly_limit_usd REAL NOT NULL,
    daily_limit_usd REAL NOT NULL DEFAULT 0,
    spent_month_usd REAL NOT NULL DEFAULT 0,
    spent_day_usd REAL NOT NULL DEFAULT 0,
    month_start TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    day_start TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    alert_80_sent BOOLEAN NOT NULL DEFAULT FALSE,
    alert_90_sent BOOLEAN NOT NULL DEFAULT FALSE,
    alert_100_sent BOOLEAN NOT NULL DEFAULT FALSE,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, scope, scope_id)
);
CREATE INDEX IF NOT EXISTS idx_budgets_tenant ON budgets(tenant_id);
