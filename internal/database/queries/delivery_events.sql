-- name: CreateDeliveryEvent :one
INSERT INTO delivery_events (delivery_id, status, note, created_by)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListDeliveryEvents :many
SELECT * FROM delivery_events
WHERE delivery_id = $1
ORDER BY created_at, id;
