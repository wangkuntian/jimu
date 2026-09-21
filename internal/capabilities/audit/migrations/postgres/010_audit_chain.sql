-- +goose Up
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS prev_hash VARCHAR(64) NOT NULL DEFAULT '';
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS entry_hash VARCHAR(64) NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS audit_chain_head (
    tenant_id BIGINT NOT NULL,
    last_hash VARCHAR(64) NOT NULL DEFAULT '',
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (tenant_id)
);

COMMENT ON COLUMN audit_logs.prev_hash IS '上一条审计的链式哈希（链首为空）';
COMMENT ON COLUMN audit_logs.entry_hash IS '本条审计的链式哈希（HMAC-SHA256 或 SHA-256）';
COMMENT ON TABLE audit_chain_head IS '审计链头（写入串行化与尾部截断检测）';
COMMENT ON COLUMN audit_chain_head.tenant_id IS '租户ID';
COMMENT ON COLUMN audit_chain_head.last_hash IS '该租户审计链最新哈希';
COMMENT ON COLUMN audit_chain_head.updated_at IS '更新时间';

-- +goose Down
DROP TABLE IF EXISTS audit_chain_head;
ALTER TABLE audit_logs DROP COLUMN IF EXISTS entry_hash;
ALTER TABLE audit_logs DROP COLUMN IF EXISTS prev_hash;
