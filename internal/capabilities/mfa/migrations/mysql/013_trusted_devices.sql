-- +goose Up
CREATE TABLE IF NOT EXISTS trusted_devices (
    id BIGINT UNSIGNED NOT NULL COMMENT '设备ID',
    tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '所属租户ID（0=未归属）',
    user_id BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '用户ID',
    token_hash VARCHAR(64) NOT NULL COMMENT '设备令牌 SHA-256 哈希',
    label VARCHAR(64) NOT NULL DEFAULT '' COMMENT '设备标签（可选，便于用户识别）',
    ip VARCHAR(64) NOT NULL DEFAULT '' COMMENT '最近登录IP（仅记录，不参与校验）',
    user_agent VARCHAR(256) NOT NULL DEFAULT '' COMMENT '最近登录 User-Agent',
    expires_at TIMESTAMP NOT NULL COMMENT '过期时间',
    last_used_at TIMESTAMP NULL COMMENT '最近使用时间',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    PRIMARY KEY (id),
    UNIQUE INDEX uk_trusted_devices_token (token_hash),
    INDEX idx_trusted_devices_user (user_id, expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='可信设备表（登录时跳过 TOTP，密码始终必需）';

-- +goose Down
DROP TABLE IF EXISTS trusted_devices;
