DROP TABLE route_items;
DROP TABLE routes;

ALTER TABLE deliveries
    DROP CONSTRAINT deliveries_coordinates_check,
    DROP COLUMN recipient_phone,
    DROP COLUMN postal_code,
    DROP COLUMN street,
    DROP COLUMN number,
    DROP COLUMN complement,
    DROP COLUMN district,
    DROP COLUMN city,
    DROP COLUMN state,
    DROP COLUMN address_reference,
    DROP COLUMN latitude,
    DROP COLUMN longitude;
