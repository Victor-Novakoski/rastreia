-- name: DeleteExpiredIdempotencyKey :exec
DELETE FROM idempotency_keys
WHERE user_id = $1 AND key = $2 AND created_at < now() - interval '24 hours';

-- ReserveIdempotencyKey returns no row when the key already exists. A
-- concurrent request with the same key waits here until the first one
-- commits or rolls back.
-- name: ReserveIdempotencyKey :one
INSERT INTO idempotency_keys (user_id, key, request_hash)
VALUES ($1, $2, $3)
ON CONFLICT (user_id, key) DO NOTHING
RETURNING *;

-- name: GetIdempotencyKey :one
SELECT * FROM idempotency_keys WHERE user_id = $1 AND key = $2;

-- name: SetIdempotencyKeyDelivery :exec
UPDATE idempotency_keys SET delivery_id = $3 WHERE user_id = $1 AND key = $2;
