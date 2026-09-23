-- Audit log (append-only)
--
-- Registro inmutable de acciones importantes: quién, qué, cuándo, dónde.
-- Solo INSERT. Nunca UPDATE ni DELETE.
--
-- Se alimenta de eventos audit.* del EventBus a través del consumer audit
-- del worker. El gateway publica los eventos (atomic TX con la acción real
-- cuando sea posible).

CREATE TABLE IF NOT EXISTS audit_log (
    id             TEXT PRIMARY KEY,
    event_id       TEXT NOT NULL,
    action         TEXT NOT NULL,
    actor_id       TEXT,
    actor_email    TEXT,
    tenant_id      TEXT,
    project_id     TEXT,
    resource_type  TEXT,
    resource_id    TEXT,
    ip             TEXT,
    user_agent     TEXT,
    request_id     TEXT,
    trace_id       TEXT,
    payload        JSONB NOT NULL DEFAULT '{}'::jsonb,
    occurred_at    TIMESTAMPTZ NOT NULL,
    recorded_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Índices para forense y dashboard
CREATE INDEX IF NOT EXISTS idx_audit_tenant_time
    ON audit_log (tenant_id, occurred_at DESC);

CREATE INDEX IF NOT EXISTS idx_audit_actor_time
    ON audit_log (actor_id, occurred_at DESC);

CREATE INDEX IF NOT EXISTS idx_audit_action_time
    ON audit_log (action, occurred_at DESC);

CREATE INDEX IF NOT EXISTS idx_audit_resource
    ON audit_log (resource_type, resource_id);

-- Unicidad por event_id: el consumer es idempotente.
-- Si el mismo evento llega 2 veces (at-least-once), no duplica.
CREATE UNIQUE INDEX IF NOT EXISTS idx_audit_event_id
    ON audit_log (event_id);

ALTER TABLE IF EXISTS audit_log ENABLE ROW LEVEL SECURITY;
