-- Índices adicionales para FinOps
-- Nota: usamos timestamp directo en vez de expression index por inmutabilidad

CREATE INDEX IF NOT EXISTS idx_usage_model 
    ON usage_records(model, timestamp DESC);
