-- +goose Up
ALTER TABLE users ADD COLUMN IF NOT EXISTS version BIGINT NOT NULL DEFAULT 0;

COMMENT ON COLUMN users.version IS '乐观锁版本号（每次更新自增）';

-- +goose Down
ALTER TABLE users DROP COLUMN IF EXISTS version;
