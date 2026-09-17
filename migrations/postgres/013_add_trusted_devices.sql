-- +goose Up
CREATE TABLE IF NOT EXISTS trusted_devices (
    id BIGINT NOT NULL,
    tenant_id BIGINT NOT NULL DEFAULT 0,
    user_id BIGINT NOT NULL DEFAULT 0,
    token_hash VARCHAR(64) NOT NULL,
    label VARCHAR(64) NOT NULL DEFAULT '',
    ip VARCHAR(64) NOT NULL DEFAULT '',
    user_agent VARCHAR(256) NOT NULL DEFAULT '',
    expires_at TIMESTAMP NOT NULL,
    last_used_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id)
);
CREATE UNIQUE INDEX IF NOT EXISTS uk_trusted_devices_token ON trusted_devices (token_hash);
CREATE INDEX IF NOT EXISTS idx_trusted_devices_user ON trusted_devices (user_id, expires_at);

COMMENT ON TABLE trusted_devices IS '可信设备表（登录时跳过 TOTP，密码始终必需）';
COMMENT ON COLUMN trusted_devices.tenant_id IS '所属租户ID（0=未归属）';
COMMENT ON COLUMN trusted_devices.user_id IS '用户ID';
COMMENT ON COLUMN trusted_devices.token_hash IS '设备令牌 SHA-256 哈希';
COMMENT ON COLUMN trusted_devices.label IS '设备标签（可选，便于用户识别）';
COMMENT ON COLUMN trusted_devices.ip IS '最近登录IP（仅记录，不参与校验）';
COMMENT ON COLUMN trusted_devices.user_agent IS '最近登录 User-Agent';
COMMENT ON COLUMN trusted_devices.expires_at IS '过期时间';
COMMENT ON COLUMN trusted_devices.last_used_at IS '最近使用时间';
COMMENT ON COLUMN trusted_devices.created_at IS '创建时间';

-- +goose Down
DROP TABLE IF EXISTS trusted_devices;
