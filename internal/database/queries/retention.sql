-- AnonymizeDeliveries erases the recipient of deliveries finished before
-- the given time, the drivers' notes, which are free text and may name
-- people, and the browsers still following them (a failed delivery keeps
-- them). Returns how many deliveries were anonymized.
-- name: AnonymizeDeliveries :one
WITH d AS (
    UPDATE deliveries SET
        recipient_name  = 'Destinatário removido',
        recipient_email = '',
        recipient_phone = '',
        address         = '',
        postal_code     = '',
        street          = '',
        number          = '',
        complement      = '',
        district        = '',
        address_reference = '',
        latitude        = NULL,
        longitude       = NULL,
        anonymized_at   = now(),
        updated_at      = now()
    WHERE anonymized_at IS NULL
      AND status IN ('delivered', 'failed')
      AND completed_at < sqlc.arg('before')::timestamptz
    RETURNING id
), e AS (
    UPDATE delivery_events SET note = NULL
    WHERE delivery_id IN (SELECT id FROM d) AND note IS NOT NULL
), s AS (
    DELETE FROM push_subscriptions
    WHERE delivery_id IN (SELECT id FROM d)
)
SELECT count(*) FROM d;

-- DeleteOldIdempotencyKeys drops the keys older than the 24 hours in which a
-- retry can still use them.
-- name: DeleteOldIdempotencyKeys :execrows
DELETE FROM idempotency_keys WHERE created_at < now() - interval '24 hours';

-- DeleteExpiredRefreshTokens drops the refresh tokens that no longer log
-- anyone in. Reuse is only checked on tokens that have not expired, so
-- nothing is lost.
-- name: DeleteExpiredRefreshTokens :execrows
DELETE FROM refresh_tokens WHERE expires_at < now();
