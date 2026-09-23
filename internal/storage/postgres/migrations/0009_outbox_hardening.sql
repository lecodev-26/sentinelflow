-- Outbox hardening
--
-- Añade updated_at para detectar eventos colgados en estado 'publishing'.
-- Si un worker muere entre el claim y el markPublished/markFailed, la fila
-- se queda en 'publishing' para siempre. El publisher, al arrancar, marca
-- como 'failed' cualquier fila 'publishing' cuyo updated_at sea más antiguo
-- que stuckThreshold (RecoverStuck).

ALTER TABLE outbox_events
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- Trigger ligero para mantener updated_at al día en cada UPDATE.
CREATE OR REPLACE FUNCTION outbox_touch_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_outbox_updated_at ON outbox_events;
CREATE TRIGGER trg_outbox_updated_at
    BEFORE UPDATE ON outbox_events
    FOR EACH ROW
    EXECUTE FUNCTION outbox_touch_updated_at();

-- Índice parcial para el barrido de stuck (solo mira publishing)
CREATE INDEX IF NOT EXISTS idx_outbox_publishing
    ON outbox_events (updated_at)
    WHERE status = 'publishing';
