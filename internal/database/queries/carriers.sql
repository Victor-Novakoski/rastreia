-- name: CreateCarrier :one
INSERT INTO carriers (name, document)
VALUES ($1, $2)
RETURNING *;

-- name: GetCarrier :one
SELECT * FROM carriers WHERE id = $1;

-- CreateCarrierWithOwner signs a carrier up with the person who runs it, in
-- one statement, so a taken e-mail leaves no carrier behind.
-- name: CreateCarrierWithOwner :one
WITH c AS (
    INSERT INTO carriers (name, document)
    VALUES (sqlc.arg('carrier_name'), sqlc.narg('document'))
    RETURNING id
)
INSERT INTO users (carrier_id, name, email, password_hash, role)
SELECT c.id, sqlc.arg('name'), sqlc.arg('email'), sqlc.arg('password_hash'), 'carrier'
FROM c
RETURNING *;
