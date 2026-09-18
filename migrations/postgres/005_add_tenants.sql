-- +goose Up
CREATE TABLE IF NOT EXISTS tenants (
    id BIGINT NOT NULL,
    code VARCHAR(64) NOT NULL,
    name VARCHAR(128) NOT NULL,
    status SMALLINT NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    PRIMARY KEY (id)
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_tenants_code ON tenants (code);
CREATE INDEX IF NOT EXISTS idx_tenants_deleted_at ON tenants (deleted_at);

ALTER TABLE users ADD COLUMN IF NOT EXISTS tenant_id BIGINT NOT NULL DEFAULT 0;
CREATE INDEX IF NOT EXISTS idx_users_tenant_id ON users (tenant_id);

ALTER TABLE roles ADD COLUMN IF NOT EXISTS tenant_id BIGINT NOT NULL DEFAULT 0;
CREATE INDEX IF NOT EXISTS idx_roles_tenant_id ON roles (tenant_id);
-- 角色名唯一性从全局调整为租户内唯一
ALTER TABLE roles DROP CONSTRAINT IF EXISTS roles_name_key;
CREATE UNIQUE INDEX IF NOT EXISTS idx_roles_tenant_name ON roles (tenant_id, name);

ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS tenant_id BIGINT NOT NULL DEFAULT 0;
CREATE INDEX IF NOT EXISTS idx_audit_logs_tenant_id ON audit_logs (tenant_id);

-- 存量数据迁入默认租户（id=1，与 kernel/tenant.DefaultTenantID 保持一致）
INSERT INTO tenants (id, code, name, status, created_at, updated_at)
VALUES (1, 'default', '默认租户', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (id) DO NOTHING;

UPDATE users SET tenant_id = 1 WHERE tenant_id = 0;
UPDATE roles SET tenant_id = 1 WHERE tenant_id = 0;

COMMENT ON TABLE tenants IS '租户表';
COMMENT ON COLUMN tenants.id IS '租户ID';
COMMENT ON COLUMN tenants.code IS '租户编码（唯一标识）';
COMMENT ON COLUMN tenants.name IS '租户名称';
COMMENT ON COLUMN tenants.status IS '状态：1-启用 0-禁用';
COMMENT ON COLUMN tenants.created_at IS '创建时间';
COMMENT ON COLUMN tenants.updated_at IS '更新时间';
COMMENT ON COLUMN tenants.deleted_at IS '删除时间（软删除）';
COMMENT ON COLUMN users.tenant_id IS '所属租户ID（0=未归属）';
COMMENT ON COLUMN roles.tenant_id IS '所属租户ID（0=未归属）';
COMMENT ON COLUMN audit_logs.tenant_id IS '所属租户ID（0=未归属）';

-- +goose Down
DROP INDEX IF EXISTS idx_audit_logs_tenant_id;
ALTER TABLE audit_logs DROP COLUMN IF EXISTS tenant_id;
DROP INDEX IF EXISTS idx_roles_tenant_name;
DROP INDEX IF EXISTS idx_roles_tenant_id;
ALTER TABLE roles DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE roles ADD CONSTRAINT roles_name_key UNIQUE (name);
DROP INDEX IF EXISTS idx_users_tenant_id;
ALTER TABLE users DROP COLUMN IF EXISTS tenant_id;
DROP TABLE IF EXISTS tenants;
