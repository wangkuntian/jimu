-- +goose Up
CREATE TABLE IF NOT EXISTS tenants (
    id BIGINT UNSIGNED NOT NULL COMMENT '租户ID',
    code VARCHAR(64) NOT NULL COMMENT '租户编码（唯一标识）',
    name VARCHAR(128) NOT NULL COMMENT '租户名称',
    status TINYINT NOT NULL DEFAULT 1 COMMENT '状态：1-启用 0-禁用',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    deleted_at TIMESTAMP NULL DEFAULT NULL COMMENT '删除时间（软删除）',
    PRIMARY KEY (id),
    UNIQUE INDEX idx_tenants_code (code),
    INDEX idx_tenants_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='租户表';

ALTER TABLE users
    ADD COLUMN tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '所属租户ID（0=未归属）' AFTER status;
CREATE INDEX idx_users_tenant_id ON users (tenant_id);

ALTER TABLE roles
    ADD COLUMN tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '所属租户ID（0=未归属）' AFTER name;
CREATE INDEX idx_roles_tenant_id ON roles (tenant_id);
-- 角色名唯一性从全局调整为租户内唯一
ALTER TABLE roles DROP INDEX name;
CREATE UNIQUE INDEX idx_roles_tenant_name ON roles (tenant_id, `name`);

-- 存量数据迁入默认租户（id=1，与 kernel/tenant.DefaultTenantID 保持一致）
INSERT IGNORE INTO tenants (id, code, name, status, created_at, updated_at)
VALUES (1, 'default', '默认租户', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);

UPDATE users SET tenant_id = 1 WHERE tenant_id = 0;
UPDATE roles SET tenant_id = 1 WHERE tenant_id = 0;

-- +goose Down
ALTER TABLE roles ADD UNIQUE INDEX name (`name`);
DROP INDEX idx_roles_tenant_name ON roles;
DROP INDEX idx_roles_tenant_id ON roles;
ALTER TABLE roles DROP COLUMN tenant_id;
DROP INDEX idx_users_tenant_id ON users;
ALTER TABLE users DROP COLUMN tenant_id;
DROP TABLE IF EXISTS tenants;
