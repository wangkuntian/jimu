-- +goose Up
CREATE TABLE IF NOT EXISTS user_mfa (
    id BIGINT UNSIGNED NOT NULL COMMENT '记录ID',
    tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '所属租户ID（0=未归属）',
    user_id BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '用户ID',
    totp_secret TEXT NULL COMMENT 'TOTP 密钥（base32，AES-GCM 密文）',
    totp_enabled TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否启用 TOTP 二次验证',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (id),
    INDEX idx_user_mfa_user (user_id),
    INDEX idx_user_mfa_tenant (tenant_id, user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户 TOTP 二次验证状态';

-- 密文原样搬迁（users.totp_secret 与 user_mfa.totp_secret 均为 AES-GCM 密文，
-- 落库格式一致，不重新加解密）；仅搬迁已启用用户，未启用的空密钥无需迁移。
-- id 复用 users.id：一个用户至多一条 MFA 记录，主键按表独立，且雪花 ID 全局唯一，
-- 不会与后续运行时生成的 user_mfa.id 冲突。
INSERT INTO user_mfa (id, tenant_id, user_id, totp_secret, totp_enabled)
    SELECT id, tenant_id, id, totp_secret, totp_enabled FROM users WHERE totp_enabled = 1;

-- 搬走后删除源列。仅当下行无加密需求时执行；迁移完成后 user 与 mfa 各自独立。
ALTER TABLE users
    DROP COLUMN totp_enabled,
    DROP COLUMN totp_secret;

-- +goose Down
ALTER TABLE users
    ADD COLUMN totp_secret TEXT NULL COMMENT 'TOTP 密钥（base32，AES-GCM 密文）' AFTER phone_hash,
    ADD COLUMN totp_enabled TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否启用 TOTP 二次验证' AFTER totp_secret;

UPDATE users u
    JOIN user_mfa m ON m.user_id = u.id
    SET u.totp_secret = m.totp_secret, u.totp_enabled = m.totp_enabled
    WHERE m.totp_enabled = 1;

DROP TABLE IF EXISTS user_mfa;
