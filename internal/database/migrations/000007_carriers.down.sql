ALTER TABLE deliveries DROP COLUMN carrier_id;
ALTER TABLE users DROP COLUMN carrier_id;
ALTER TABLE users DROP CONSTRAINT users_role_check;
UPDATE users SET role = 'admin' WHERE role = 'carrier';
ALTER TABLE users ADD CONSTRAINT users_role_check CHECK (role IN ('admin', 'driver'));
DROP TABLE carriers;
