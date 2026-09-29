-- +goose Up
ALTER TABLE users ADD COLUMN version BIGINT NOT NULL DEFAULT 0 COMMENT '乐观锁版本号（每次更新自增）';

-- +goose Down
ALTER TABLE users DROP COLUMN version;
