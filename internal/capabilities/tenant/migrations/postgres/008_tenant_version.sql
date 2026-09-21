-- +goose Up
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS version BIGINT NOT NULL DEFAULT 0;

COMMENT ON COLUMN tenants.version IS '乐观锁版本号（每次更新自增）';

-- +goose Down
ALTER TABLE tenants DROP COLUMN IF EXISTS version;
