-- EnsureRoute returns the driver's route for the day, creating it on first use.
-- name: EnsureRoute :one
INSERT INTO routes (carrier_id, driver_id, route_date)
VALUES ($1, $2, $3)
ON CONFLICT (driver_id, route_date) DO UPDATE SET route_date = excluded.route_date
RETURNING *;

-- ListRouteItems lists the route's packages in order. A package the carrier
-- gave to another driver after it was scanned leaves the route, so the first
-- driver no longer sees the recipient.
-- name: ListRouteItems :many
SELECT sqlc.embed(d), ri.position
FROM route_items ri
JOIN routes r ON r.id = ri.route_id
JOIN deliveries d ON d.id = ri.delivery_id AND d.driver_id = r.driver_id
WHERE ri.route_id = $1
ORDER BY ri.position, ri.added_at;

-- AddRouteItem puts the delivery at the end of the route; adding it twice
-- changes nothing and affects no rows.
-- name: AddRouteItem :execrows
INSERT INTO route_items (route_id, delivery_id, position)
SELECT sqlc.arg('route_id'), sqlc.arg('delivery_id'),
       coalesce(max(position), 0) + 1
FROM route_items WHERE route_id = sqlc.arg('route_id')
ON CONFLICT (route_id, delivery_id) DO NOTHING;

-- name: RemoveRouteItem :execrows
DELETE FROM route_items WHERE route_id = $1 AND delivery_id = $2;

-- SetRoutePositions numbers the route's deliveries in the order of the ids
-- given, from 1.
-- name: SetRoutePositions :exec
UPDATE route_items ri SET position = u.position
FROM unnest(sqlc.arg('delivery_ids')::bigint[]) WITH ORDINALITY AS u(delivery_id, position)
WHERE ri.route_id = sqlc.arg('route_id') AND ri.delivery_id = u.delivery_id;

-- ClaimDelivery assigns a delivery with no driver to the driver who scanned
-- it; one that already has a driver is left alone.
-- name: ClaimDelivery :one
UPDATE deliveries SET driver_id = sqlc.arg('driver_id'), updated_at = now()
WHERE id = sqlc.arg('id') AND driver_id IS NULL
RETURNING *;
