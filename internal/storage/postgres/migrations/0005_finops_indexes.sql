-- Índices adicionales para FinOps

CREATE INDEX IF NOT EXISTS idx_usage_tenant_day 
    ON usage_records(tenant_id, (timestamp::date));

CREATE INDEX IF NOT EXISTS idx_usage_model 
    ON usage_records(model, timestamp DESC);
