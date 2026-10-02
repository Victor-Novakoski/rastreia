-- delivery_events doubles as the outbox for notifications: the event and the
-- "to publish" mark are written in the same transaction, so a status change
-- is never saved without its notification, or the other way around. The
-- relay in the API publishes rows with published_at NULL to RabbitMQ.
ALTER TABLE delivery_events ADD COLUMN published_at TIMESTAMPTZ;

-- Events from before notifications existed must not send e-mails now.
UPDATE delivery_events SET published_at = created_at;

CREATE INDEX delivery_events_unpublished_idx ON delivery_events (id) WHERE published_at IS NULL;
