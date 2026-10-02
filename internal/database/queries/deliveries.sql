-- name: CreateDelivery :one
INSERT INTO deliveries (carrier_id, tracking_code, recipient_name, recipient_email, address, driver_id)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetDelivery :one
SELECT * FROM deliveries WHERE id = $1;

-- name: ListDeliveries :many
SELECT * FROM deliveries
WHERE carrier_id = sqlc.arg('carrier_id')
  AND (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status')::text)
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

-- name: ListDriverDeliveries :many
SELECT * FROM deliveries
WHERE driver_id = sqlc.arg('driver_id')::bigint
  AND (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status')::text)
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: GetDeliveryByTrackingCode :one
SELECT * FROM deliveries WHERE tracking_code = $1;

-- SetDeliveryStatus only changes the row if the status is still the one the
-- caller saw, so two concurrent events cannot both apply.
-- name: SetDeliveryStatus :one
UPDATE deliveries SET
    status       = sqlc.arg('status'),
    completed_at = sqlc.narg('completed_at'),
    updated_at   = now()
WHERE id = sqlc.arg('id') AND status = sqlc.arg('from_status')
RETURNING *;

-- CountDeliveriesByStatus feeds the carrier's dashboard. since limits the
-- count to deliveries created from that moment on.
-- name: CountDeliveriesByStatus :many
SELECT status, count(*) AS total FROM deliveries
WHERE carrier_id = sqlc.arg('carrier_id')
  AND created_at >= sqlc.arg('since')::timestamptz
GROUP BY status;

-- name: CountUnassignedDeliveries :one
SELECT count(*) FROM deliveries
WHERE carrier_id = $1 AND driver_id IS NULL AND status = 'pending';
