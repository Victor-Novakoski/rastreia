-- name: CreateDelivery :one
INSERT INTO deliveries (tracking_code, recipient_name, recipient_email, address, driver_id)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetDelivery :one
SELECT * FROM deliveries WHERE id = $1;

-- name: ListDeliveries :many
SELECT * FROM deliveries
WHERE sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status')::text
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: UpdateDelivery :one
UPDATE deliveries SET
    recipient_name  = coalesce(sqlc.narg('recipient_name'), recipient_name),
    recipient_email = coalesce(sqlc.narg('recipient_email'), recipient_email),
    address         = coalesce(sqlc.narg('address'), address),
    driver_id       = coalesce(sqlc.narg('driver_id'), driver_id),
    updated_at      = now()
WHERE id = sqlc.arg('id')
RETURNING *;
