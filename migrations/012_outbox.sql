-- Outbox: menu-change events are written here and relayed to Kafka
-- (pkg/outbox), so they survive a Kafka outage.
CREATE TABLE IF NOT EXISTS outbox (
    id           BIGSERIAL PRIMARY KEY,
    topic        TEXT        NOT NULL,
    key          TEXT        NOT NULL,
    payload      JSONB       NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at TIMESTAMPTZ,
    attempts     INT         NOT NULL DEFAULT 0,
    last_error   TEXT
);

CREATE INDEX IF NOT EXISTS idx_outbox_pending ON outbox (id) WHERE published_at IS NULL;
