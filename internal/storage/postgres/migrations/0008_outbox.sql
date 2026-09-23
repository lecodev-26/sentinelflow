-- Transactional Outbox
--
-- Garantiza que los eventos se persisten en la MISMA transacción que el
-- cambio de negocio. Un worker los publica después al EventBus.
--
-- Problema que resuelve:
--   BEGIN TX
--     INSERT usage_record    OK
--   COMMIT                   OK
--   PUBLISH event            FAIL  <-- evento perdido para siempre
--
-- Con outbox:
--   BEGIN TX
--     INSERT usage_record    OK
--     INSERT outbox_event    OK
--   COMMIT                   OK
--   Worker lee outbox -> publica al bus -> marca published
--
-- Garantía: at-least-once. La idempotencia es responsabilidad del consumer.
-- Alineado con RULE 07 de ARCHITECTURE.md: "Every distributed event is idempotent".

CREATE TABLE IF NOT EXISTS outbox_events (
    -- Identidad del evento (coincide con events.Event.ID)
    event_id        TEXT PRIMARY KEY,

    -- Clasificación
    event_type      TEXT NOT NULL,

    -- Correlación multi-tenant (alineado con events.Event)
    tenant_id       TEXT,
    project_id      TEXT,
    user_id         TEXT,
    request_id      TEXT,
    trace_id        TEXT,

    -- Payload libre (events.Event.Payload)
    payload         JSONB NOT NULL DEFAULT '{}'::jsonb,

    -- Control del publisher
    status          TEXT NOT NULL DEFAULT 'pending',
    attempts        INT  NOT NULL DEFAULT 0,
    last_error      TEXT,
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Timestamps
    occurred_at     TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at    TIMESTAMPTZ,

    CONSTRAINT outbox_status_check
        CHECK (status IN ('pending','publishing','published','failed','dead')),
    CONSTRAINT outbox_attempts_check
        CHECK (attempts >= 0)
);

-- Índice crítico del publisher.
-- Parcial: solo mira lo pendiente/fallido. En estado estable, el 99% de las
-- filas están 'published' y NO entran en este índice.
CREATE INDEX IF NOT EXISTS idx_outbox_pending
    ON outbox_events (next_attempt_at)
    WHERE status IN ('pending','failed');

-- Auditoría por tenant (dashboard, debugging, forense)
CREATE INDEX IF NOT EXISTS idx_outbox_tenant_created
    ON outbox_events (tenant_id, created_at DESC);

-- Auditoría por tipo de evento (útil para replays selectivos)
CREATE INDEX IF NOT EXISTS idx_outbox_type_created
    ON outbox_events (event_type, created_at DESC);

-- Correlación de trazas (V4.5 observabilidad)
CREATE INDEX IF NOT EXISTS idx_outbox_trace
    ON outbox_events (trace_id)
    WHERE trace_id IS NOT NULL;

-- RLS (consistente con 0006_enable_rls.sql)
ALTER TABLE IF EXISTS outbox_events ENABLE ROW LEVEL SECURITY;
