-- Analytics daily aggregates
--
-- Tabla de agregados por tenant + día, actualizada por el consumer
-- analytics del worker a partir de eventos usage.recorded del bus.
--
-- Permite consultar "coste del día X del tenant Y" sin escanear
-- toda la tabla usage_records. Idempotente vía UPSERT.

CREATE TABLE IF NOT EXISTS analytics_daily (
    tenant_id      TEXT NOT NULL,
    day            DATE NOT NULL,
    requests       BIGINT NOT NULL DEFAULT 0,
    input_tokens   BIGINT NOT NULL DEFAULT 0,
    output_tokens  BIGINT NOT NULL DEFAULT 0,
    total_tokens   BIGINT NOT NULL DEFAULT 0,
    cost_usd       NUMERIC(16, 6) NOT NULL DEFAULT 0,
    errors         BIGINT NOT NULL DEFAULT 0,
    last_updated   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, day)
);

CREATE INDEX IF NOT EXISTS idx_analytics_daily_day
    ON analytics_daily (day DESC);

ALTER TABLE IF EXISTS analytics_daily ENABLE ROW LEVEL SECURITY;
