-- Webhooks: notificaciones salientes a clientes
--
-- Cliente registra una URL + secret + lista de event_types.
-- Cuando un evento de esos tipos entra al bus, el consumer webhook
-- encola una delivery y el dispatcher la envía con HMAC.
--
-- Reintentos con backoff. Firma HMAC-SHA256 en header X-SF-Signature.

CREATE TABLE IF NOT EXISTS webhooks (
    id             TEXT PRIMARY KEY,
    tenant_id      TEXT NOT NULL,
    url            TEXT NOT NULL,
    secret         TEXT NOT NULL,
    event_types    JSONB NOT NULL DEFAULT '[]'::jsonb,
    description    TEXT,
    active         BOOLEAN NOT NULL DEFAULT true,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_webhooks_tenant
    ON webhooks (tenant_id);

CREATE INDEX IF NOT EXISTS idx_webhooks_active
    ON webhooks (active) WHERE active = true;

CREATE TABLE IF NOT EXISTS webhook_deliveries (
    id             TEXT PRIMARY KEY,
    webhook_id     TEXT NOT NULL REFERENCES webhooks(id) ON DELETE CASCADE,
    event_id       TEXT NOT NULL,
    event_type     TEXT NOT NULL,
    payload        JSONB NOT NULL,
    status         TEXT NOT NULL DEFAULT 'pending',
    attempts       INT NOT NULL DEFAULT 0,
    last_error     TEXT,
    response_code  INT,
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    delivered_at   TIMESTAMPTZ,
    CONSTRAINT webhook_delivery_status_check
        CHECK (status IN ('pending','delivering','delivered','failed','dead'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_webhook_deliveries_dedup
    ON webhook_deliveries (webhook_id, event_id);

CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_pending
    ON webhook_deliveries (next_attempt_at)
    WHERE status IN ('pending','failed');

CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_webhook_time
    ON webhook_deliveries (webhook_id, created_at DESC);

ALTER TABLE IF EXISTS webhooks ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS webhook_deliveries ENABLE ROW LEVEL SECURITY;
