-- name: CreateUser :one
INSERT INTO users (carrier_id, name, email, password_hash, role)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: ListCarrierUsersByRole :many
SELECT * FROM users WHERE carrier_id = $1 AND role = $2 ORDER BY name;
