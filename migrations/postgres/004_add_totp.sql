-- +goose Up
ALTER TABLE users
    ADD COLUMN totp_secret TEXT NULL,
    ADD COLUMN totp_enabled SMALLINT NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE users
    DROP COLUMN totp_enabled,
    DROP COLUMN totp_secret;