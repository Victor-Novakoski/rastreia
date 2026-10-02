-- The address is kept in parts so the carrier fills it from the CEP and the
-- driver gets a phone and a reference. address stays as the full text, built
-- from the parts; deliveries created before keep only that text.
ALTER TABLE deliveries
    ADD COLUMN recipient_phone   TEXT NOT NULL DEFAULT '',
    ADD COLUMN postal_code       TEXT NOT NULL DEFAULT '',
    ADD COLUMN street            TEXT NOT NULL DEFAULT '',
    ADD COLUMN number            TEXT NOT NULL DEFAULT '',
    ADD COLUMN complement        TEXT NOT NULL DEFAULT '',
    ADD COLUMN district          TEXT NOT NULL DEFAULT '',
    ADD COLUMN city              TEXT NOT NULL DEFAULT '',
    ADD COLUMN state             TEXT NOT NULL DEFAULT '',
    ADD COLUMN address_reference TEXT NOT NULL DEFAULT '',
    ADD COLUMN latitude          DOUBLE PRECISION,
    ADD COLUMN longitude         DOUBLE PRECISION,
    ADD CONSTRAINT deliveries_coordinates_check
        CHECK ((latitude IS NULL) = (longitude IS NULL));

-- A driver's route for one day: the packages loaded by scanning their codes,
-- in the order the driver will deliver them.
CREATE TABLE routes (
    id         BIGSERIAL PRIMARY KEY,
    carrier_id BIGINT      NOT NULL REFERENCES carriers (id),
    driver_id  BIGINT      NOT NULL REFERENCES users (id),
    route_date DATE        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (driver_id, route_date)
);

CREATE TABLE route_items (
    route_id    BIGINT      NOT NULL REFERENCES routes (id) ON DELETE CASCADE,
    delivery_id BIGINT      NOT NULL REFERENCES deliveries (id),
    position    INT         NOT NULL,
    added_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (route_id, delivery_id)
);

CREATE INDEX route_items_delivery_id_idx ON route_items (delivery_id);
