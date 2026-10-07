-- name: CreateDelivery :one
INSERT INTO deliveries (
    carrier_id, tracking_code, recipient_name, recipient_email, recipient_phone,
    address, postal_code, street, number, complement, district, city, state,
    address_reference, latitude, longitude, driver_id
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
RETURNING *;

-- name: GetDelivery :one
SELECT * FROM deliveries WHERE id = $1;

-- ListDeliveries lists the carrier's deliveries, newest first. search, when
-- given, matches part of the tracking code or of the recipient's name or
-- e-mail. The service sends it in lower case, without accents and with the
-- LIKE wildcards escaped; translate drops the same accents from the name
-- (the letters of foldAccents in internal/delivery), so "joao" finds "João".
-- name: ListDeliveries :many
SELECT * FROM deliveries
WHERE carrier_id = sqlc.arg('carrier_id')
  AND (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status')::text)
  AND (sqlc.narg('search')::text IS NULL
       OR lower(tracking_code) LIKE '%' || sqlc.narg('search')::text || '%'
       OR lower(translate(recipient_name,
                'ÁÀÂÃÄÉÈÊËÍÌÎÏÓÒÔÕÖÚÙÛÜÇÑáàâãäéèêëíìîïóòôõöúùûüçñ',
                'aaaaaeeeeiiiiooooouuuucnaaaaaeeeeiiiiooooouuuucn'))
          LIKE '%' || sqlc.narg('search')::text || '%'
       OR lower(recipient_email) LIKE '%' || sqlc.narg('search')::text || '%')
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- UpdateDelivery writes every recipient and address field: the service
-- merges the change into the current delivery first. The driver is only
-- changed when sent. An anonymized delivery is left alone, or an edit racing
-- the retention job would write the erased data back.
-- name: UpdateDelivery :one
UPDATE deliveries SET
    recipient_name    = sqlc.arg('recipient_name'),
    recipient_email   = sqlc.arg('recipient_email'),
    recipient_phone   = sqlc.arg('recipient_phone'),
    address           = sqlc.arg('address'),
    postal_code       = sqlc.arg('postal_code'),
    street            = sqlc.arg('street'),
    number            = sqlc.arg('number'),
    complement        = sqlc.arg('complement'),
    district          = sqlc.arg('district'),
    city              = sqlc.arg('city'),
    state             = sqlc.arg('state'),
    address_reference = sqlc.arg('address_reference'),
    latitude          = sqlc.narg('latitude'),
    longitude         = sqlc.narg('longitude'),
    driver_id         = coalesce(sqlc.narg('driver_id'), driver_id),
    updated_at        = now()
WHERE id = sqlc.arg('id') AND anonymized_at IS NULL
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
-- caller saw, so two concurrent events cannot both apply, and if the
-- recipient's data was not erased in the meantime.
-- name: SetDeliveryStatus :one
UPDATE deliveries SET
    status       = sqlc.arg('status'),
    completed_at = sqlc.narg('completed_at'),
    updated_at   = now()
WHERE id = sqlc.arg('id') AND status = sqlc.arg('from_status') AND anonymized_at IS NULL
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
