-- History of status changes. created_by is NULL for events written by the
-- system (the backfill below).
CREATE TABLE delivery_events (
    id          BIGSERIAL PRIMARY KEY,
    delivery_id BIGINT      NOT NULL REFERENCES deliveries (id) ON DELETE CASCADE,
    status      TEXT        NOT NULL
                CHECK (status IN ('pending', 'picked_up', 'in_transit', 'delivered', 'failed')),
    note        TEXT,
    created_by  BIGINT      REFERENCES users (id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX delivery_events_delivery_id_idx ON delivery_events (delivery_id, created_at);

-- When the delivery last reached delivered or failed; the public tracking
-- link expires 30 days after it. Cleared when a failed delivery goes back
-- in transit.
ALTER TABLE deliveries ADD COLUMN completed_at TIMESTAMPTZ;

INSERT INTO delivery_events (delivery_id, status, created_at)
SELECT id, status, created_at FROM deliveries;

-- Lets a client retry POST /deliveries without creating the delivery twice.
CREATE TABLE idempotency_keys (
    user_id      BIGINT      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    key          TEXT        NOT NULL,
    request_hash TEXT        NOT NULL,
    delivery_id  BIGINT      REFERENCES deliveries (id) ON DELETE CASCADE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, key)
);
