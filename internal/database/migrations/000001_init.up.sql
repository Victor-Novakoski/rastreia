CREATE TABLE users (
    id            BIGSERIAL PRIMARY KEY,
    name          TEXT        NOT NULL,
    email         TEXT        NOT NULL UNIQUE,
    password_hash TEXT        NOT NULL,
    role          TEXT        NOT NULL CHECK (role IN ('admin', 'driver')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE deliveries (
    id              BIGSERIAL PRIMARY KEY,
    tracking_code   TEXT        NOT NULL UNIQUE,
    recipient_name  TEXT        NOT NULL,
    recipient_email TEXT        NOT NULL,
    address         TEXT        NOT NULL,
    status          TEXT        NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending', 'picked_up', 'in_transit', 'delivered', 'failed')),
    driver_id       BIGINT      REFERENCES users (id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX deliveries_driver_id_idx ON deliveries (driver_id);
CREATE INDEX deliveries_status_idx ON deliveries (status);
