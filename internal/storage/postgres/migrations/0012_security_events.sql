-- Security events (findings de detectors)
--
-- Almacena detecciones de: PII, secrets, prompt injection, SSRF, policy violations.
-- Alimentado por el consumer security del worker a partir de eventos
-- security.* del bus.
--
-- El índice único por event_id garantiza idempotencia (bus at-least-once).

CREATE TABLE IF NOT EXISTS security_events (
    id             TEXT PRIMARY KEY,
    event_id       TEXT NOT NULL,
    kind           TEXT NOT NULL,
    severity       TEXT NOT NULL DEFAULT 'medium',
    tenant_id      TEXT,
    project_id     TEXT,
    user_id        TEXT,
    request_id     TEXT,
    trace_id       TEXT,
    detector       TEXT,
    rule           TEXT,
    snippet        TEXT,
    action         TEXT,
    payload        JSONB NOT NULL DEFAULT '{}'::jsonb,
    occurred_at    TIMESTAMPTZ NOT NULL,
    recorded_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_security_events_event_id
    ON security_events (event_id);

CREATE INDEX IF NOT EXISTS idx_security_events_tenant_time
    ON security_events (tenant_id, occurred_at DESC);

CREATE INDEX IF NOT EXISTS idx_security_events_kind_time
    ON security_events (kind, occurred_at DESC);

CREATE INDEX IF NOT EXISTS idx_security_events_severity
    ON security_events (severity, occurred_at DESC);

ALTER TABLE IF EXISTS security_events ENABLE ROW LEVEL SECURITY;
