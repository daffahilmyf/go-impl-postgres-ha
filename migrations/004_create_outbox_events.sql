-- +goose Up
CREATE TABLE IF NOT EXISTS outbox_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_type TEXT NOT NULL,
    aggregate_id UUID NOT NULL,
    event_type TEXT NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    locked_at TIMESTAMPTZ NULL,
    processed_at TIMESTAMPTZ NULL,
    attempts INT NOT NULL DEFAULT 0,
    last_error TEXT NULL
);

CREATE INDEX IF NOT EXISTS idx_outbox_events_unprocessed ON outbox_events (processed_at) WHERE processed_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_outbox_events_locked ON outbox_events (locked_at);
CREATE INDEX IF NOT EXISTS idx_outbox_events_created_at ON outbox_events (created_at);

-- +goose Down
DROP TABLE IF EXISTS outbox_events;
