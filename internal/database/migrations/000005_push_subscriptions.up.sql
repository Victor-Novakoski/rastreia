-- Browsers that asked for push notifications about a delivery. The
-- recipient has no account, so a subscription belongs to the delivery and
-- is removed with it, when the push service says it expired, or once the
-- delivery is delivered.
CREATE TABLE push_subscriptions (
    id          BIGSERIAL PRIMARY KEY,
    delivery_id BIGINT      NOT NULL REFERENCES deliveries (id) ON DELETE CASCADE,
    endpoint    TEXT        NOT NULL,
    p256dh      TEXT        NOT NULL,
    auth        TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (delivery_id, endpoint)
);
