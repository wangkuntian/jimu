-- +goose Up
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS tenant_id BIGINT NOT NULL DEFAULT 0;
CREATE INDEX IF NOT EXISTS idx_audit_logs_tenant_id ON audit_logs (tenant_id);

COMMENT ON COLUMN audit_logs.tenant_id IS '所属租户ID（0=未归属）';

-- +goose Down
DROP INDEX IF EXISTS idx_audit_logs_tenant_id;
ALTER TABLE audit_logs DROP COLUMN IF EXISTS tenant_id;
