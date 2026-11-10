-- +goose Up
ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS tenant_id BIGINT NOT NULL DEFAULT 0;
CREATE INDEX IF NOT EXISTS idx_api_keys_tenant_id ON api_keys (tenant_id);

-- 存量 Key 迁入默认租户（id=1，与 platform/tenant.DefaultTenantID 保持一致）
UPDATE api_keys SET tenant_id = 1 WHERE tenant_id = 0;

COMMENT ON COLUMN api_keys.tenant_id IS '所属租户ID（0=未归属）';

-- +goose Down
DROP INDEX IF EXISTS idx_api_keys_tenant_id;
ALTER TABLE api_keys DROP COLUMN IF EXISTS tenant_id;
