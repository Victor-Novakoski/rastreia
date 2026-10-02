-- Each carrier (transportadora) is a tenant: its users, drivers and
-- deliveries are only visible inside it. document is the CNPJ, optional.
CREATE TABLE carriers (
    id         BIGSERIAL PRIMARY KEY,
    name       TEXT        NOT NULL,
    document   TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX carriers_document_idx ON carriers (document) WHERE document IS NOT NULL;

-- Data from before carriers existed belongs to a single carrier.
INSERT INTO carriers (name)
SELECT 'Minha transportadora'
WHERE EXISTS (SELECT 1 FROM users) OR EXISTS (SELECT 1 FROM deliveries);

-- The "admin" role becomes "carrier": the person who runs a carrier.
ALTER TABLE users DROP CONSTRAINT users_role_check;
UPDATE users SET role = 'carrier' WHERE role = 'admin';
ALTER TABLE users ADD CONSTRAINT users_role_check CHECK (role IN ('carrier', 'driver'));

ALTER TABLE users ADD COLUMN carrier_id BIGINT REFERENCES carriers (id);
UPDATE users SET carrier_id = (SELECT min(id) FROM carriers);
ALTER TABLE users ALTER COLUMN carrier_id SET NOT NULL;
CREATE INDEX users_carrier_id_idx ON users (carrier_id, role);

ALTER TABLE deliveries ADD COLUMN carrier_id BIGINT REFERENCES carriers (id);
UPDATE deliveries SET carrier_id = (SELECT min(id) FROM carriers);
ALTER TABLE deliveries ALTER COLUMN carrier_id SET NOT NULL;
CREATE INDEX deliveries_carrier_id_idx ON deliveries (carrier_id, created_at DESC);
