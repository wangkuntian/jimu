-- +goose Up
ALTER TABLE users
    ADD COLUMN totp_secret TEXT NULL COMMENT 'TOTP 密钥（base32，AES-GCM 密文）' AFTER phone_hash,
    ADD COLUMN totp_enabled TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否启用 TOTP 二次验证' AFTER totp_secret;

-- +goose Down
ALTER TABLE users
    DROP COLUMN totp_enabled,
    DROP COLUMN totp_secret;