-- +goose Up
ALTER TABLE audit_logs ADD COLUMN prev_hash VARCHAR(64) NOT NULL DEFAULT '' COMMENT '上一条审计的链式哈希（链首为空）';
ALTER TABLE audit_logs ADD COLUMN entry_hash VARCHAR(64) NOT NULL DEFAULT '' COMMENT '本条审计的链式哈希（HMAC-SHA256 或 SHA-256）';

CREATE TABLE IF NOT EXISTS audit_chain_head (
    tenant_id BIGINT UNSIGNED NOT NULL COMMENT '租户ID',
    last_hash VARCHAR(64) NOT NULL DEFAULT '' COMMENT '该租户审计链最新哈希',
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (tenant_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='审计链头（写入串行化与尾部截断检测）';

-- +goose Down
DROP TABLE IF EXISTS audit_chain_head;
ALTER TABLE audit_logs DROP COLUMN entry_hash;
ALTER TABLE audit_logs DROP COLUMN prev_hash;
