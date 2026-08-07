-- +goose Up
ALTER TABLE users ADD COLUMN tenant_id VARCHAR(64) NOT NULL DEFAULT '' COMMENT '租户 ID' AFTER status;
CREATE INDEX idx_users_tenant_id ON users (tenant_id);

ALTER TABLE audit_logs ADD COLUMN tenant_id VARCHAR(64) NOT NULL DEFAULT '' COMMENT '租户 ID' AFTER admin_name;

-- +goose Down
ALTER TABLE audit_logs DROP COLUMN tenant_id;
DROP INDEX idx_users_tenant_id ON users;
ALTER TABLE users DROP COLUMN tenant_id;
