-- +goose Up
ALTER TABLE users ADD COLUMN version BIGINT NOT NULL DEFAULT 0 COMMENT '乐观锁版本号（每次更新自增）';
ALTER TABLE roles ADD COLUMN version BIGINT NOT NULL DEFAULT 0 COMMENT '乐观锁版本号（每次更新自增）';
ALTER TABLE tenants ADD COLUMN version BIGINT NOT NULL DEFAULT 0 COMMENT '乐观锁版本号（每次更新自增）';

-- +goose Down
ALTER TABLE users DROP COLUMN version;
ALTER TABLE roles DROP COLUMN version;
ALTER TABLE tenants DROP COLUMN version;
