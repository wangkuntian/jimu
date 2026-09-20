-- +goose Up
ALTER TABLE api_keys
    ADD COLUMN tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '所属租户ID（0=未归属）' AFTER name;
CREATE INDEX idx_api_keys_tenant_id ON api_keys (tenant_id);

-- 存量 Key 迁入默认租户（id=1，与 kernel/tenant.DefaultTenantID 保持一致）
UPDATE api_keys SET tenant_id = 1 WHERE tenant_id = 0;

-- +goose Down
DROP INDEX idx_api_keys_tenant_id ON api_keys;
ALTER TABLE api_keys DROP COLUMN tenant_id;
