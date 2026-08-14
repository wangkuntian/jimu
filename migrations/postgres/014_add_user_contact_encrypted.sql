-- +goose Up
ALTER TABLE users ADD COLUMN email TEXT;
ALTER TABLE users ADD COLUMN email_hash VARCHAR(64);
ALTER TABLE users ADD COLUMN phone VARCHAR(255);
ALTER TABLE users ADD COLUMN phone_hash VARCHAR(64);

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_hash ON users (email_hash);
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_phone_hash ON users (phone_hash);

-- +goose Down
DROP INDEX IF EXISTS idx_users_phone_hash;
DROP INDEX IF EXISTS idx_users_email_hash;
ALTER TABLE users DROP COLUMN phone_hash;
ALTER TABLE users DROP COLUMN phone;
ALTER TABLE users DROP COLUMN email_hash;
ALTER TABLE users DROP COLUMN email;
