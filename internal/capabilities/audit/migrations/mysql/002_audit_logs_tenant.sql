-- +goose Up
ALTER TABLE audit_logs
    ADD COLUMN tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '所属租户ID（0=未归属）' AFTER user_id;
CREATE INDEX idx_audit_logs_tenant_id ON audit_logs (tenant_id);

-- +goose Down
DROP INDEX idx_audit_logs_tenant_id ON audit_logs;
ALTER TABLE audit_logs DROP COLUMN tenant_id;
