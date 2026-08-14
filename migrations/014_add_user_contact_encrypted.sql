-- +goose Up
ALTER TABLE users
  ADD COLUMN email TEXT NULL COMMENT '邮箱（AES-GCM 密文）' AFTER username,
  ADD COLUMN email_hash VARCHAR(64) NULL COMMENT '邮箱盲索引（HMAC-SHA256，精确查询与唯一约束）' AFTER email,
  ADD COLUMN phone VARCHAR(255) NULL COMMENT '手机号（AES-GCM 密文）' AFTER email_hash,
  ADD COLUMN phone_hash VARCHAR(64) NULL COMMENT '手机号盲索引（HMAC-SHA256，精确查询与唯一约束）' AFTER phone,
  ADD UNIQUE INDEX idx_users_email_hash (email_hash),
  ADD UNIQUE INDEX idx_users_phone_hash (phone_hash);

-- +goose Down
ALTER TABLE users
  DROP INDEX idx_users_phone_hash,
  DROP INDEX idx_users_email_hash,
  DROP COLUMN phone_hash,
  DROP COLUMN phone,
  DROP COLUMN email_hash,
  DROP COLUMN email;
