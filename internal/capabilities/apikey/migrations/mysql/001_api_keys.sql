-- +goose Up
CREATE TABLE IF NOT EXISTS api_keys (
    id BIGINT UNSIGNED NOT NULL COMMENT 'API Key ID',
    name VARCHAR(64) NOT NULL COMMENT 'Key 名称',
    key_prefix VARCHAR(16) NOT NULL COMMENT 'Key 前缀（用于识别）',
    key_hash VARCHAR(64) NOT NULL COMMENT 'SHA-256 哈希',
    scopes TEXT COMMENT '权限范围（JSON 数组）',
    enabled TINYINT(1) NOT NULL DEFAULT 1 COMMENT '是否启用',
    expires_at TIMESTAMP NULL COMMENT '过期时间',
    last_used TIMESTAMP NULL COMMENT '最后使用时间',
    use_count BIGINT NOT NULL DEFAULT 0 COMMENT '使用次数',
    created_by BIGINT UNSIGNED COMMENT '创建者用户 ID',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    PRIMARY KEY (id),
    INDEX idx_key_hash (key_hash),
    INDEX idx_created_by (created_by)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='API 密钥表';

-- +goose Down
DROP TABLE IF EXISTS api_keys;
