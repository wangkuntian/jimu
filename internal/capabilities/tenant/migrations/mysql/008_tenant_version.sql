-- +goose Up
ALTER TABLE tenants ADD COLUMN version BIGINT NOT NULL DEFAULT 0 COMMENT '乐观锁版本号（每次更新自增）';

-- +goose Down
ALTER TABLE tenants DROP COLUMN version;
