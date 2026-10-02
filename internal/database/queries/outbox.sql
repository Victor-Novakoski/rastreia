-- ClaimUnpublishedEvents locks the next events to publish. SKIP LOCKED lets
-- several API instances run the relay without sending an event twice.
-- name: ClaimUnpublishedEvents :many
SELECT e.id, e.status, e.created_at, d.tracking_code, d.recipient_name, d.recipient_email
FROM delivery_events e
JOIN deliveries d ON d.id = e.delivery_id
WHERE e.published_at IS NULL
ORDER BY e.id
LIMIT sqlc.arg('limit')
FOR UPDATE OF e SKIP LOCKED;

-- name: MarkEventsPublished :exec
UPDATE delivery_events SET published_at = now() WHERE id = ANY(sqlc.arg('ids')::bigint[]);

-- SkipStaleEvents drops notifications nobody wants anymore, such as the ones
-- piled up while RabbitMQ was off.
-- name: SkipStaleEvents :execrows
UPDATE delivery_events SET published_at = now()
WHERE published_at IS NULL AND created_at < sqlc.arg('before');
