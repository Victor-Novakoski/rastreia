-- AnonymizeDeliveries erases the recipient of deliveries finished before
-- the given time, and the drivers' notes, which are free text and may name
-- people. Returns how many deliveries were anonymized.
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
)
SELECT count(*) FROM d;
