-- name: UpsertPushSubscription :exec
INSERT INTO push_subscriptions (delivery_id, endpoint, p256dh, auth)
VALUES ($1, $2, $3, $4)
ON CONFLICT (delivery_id, endpoint) DO UPDATE SET p256dh = excluded.p256dh, auth = excluded.auth;

-- name: CountPushSubscriptions :one
SELECT count(*) FROM push_subscriptions WHERE delivery_id = $1;

-- name: DeletePushSubscription :exec
DELETE FROM push_subscriptions WHERE delivery_id = $1 AND endpoint = $2;

-- name: ListPushSubscriptionsByCode :many
SELECT s.* FROM push_subscriptions s
JOIN deliveries d ON d.id = s.delivery_id
WHERE d.tracking_code = $1
ORDER BY s.id;

-- name: DeletePushSubscriptionByID :exec
DELETE FROM push_subscriptions WHERE id = $1;

-- name: DeletePushSubscriptionsByCode :exec
DELETE FROM push_subscriptions s
USING deliveries d
WHERE d.id = s.delivery_id AND d.tracking_code = $1;
