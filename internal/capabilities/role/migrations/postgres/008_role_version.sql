-- +goose Up
ALTER TABLE roles ADD COLUMN IF NOT EXISTS version BIGINT NOT NULL DEFAULT 0;

COMMENT ON COLUMN roles.version IS '乐观锁版本号（每次更新自增）';

-- +goose Down
ALTER TABLE roles DROP COLUMN IF EXISTS version;
