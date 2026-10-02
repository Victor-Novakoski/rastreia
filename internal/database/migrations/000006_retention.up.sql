-- When the recipient's personal data was erased (LGPD). The delivery and its
-- status history stay for the carrier's reports.
ALTER TABLE deliveries ADD COLUMN anonymized_at TIMESTAMPTZ;

CREATE INDEX deliveries_to_anonymize_idx ON deliveries (completed_at)
    WHERE anonymized_at IS NULL AND completed_at IS NOT NULL;
