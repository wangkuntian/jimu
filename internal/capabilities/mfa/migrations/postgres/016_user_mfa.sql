-- +goose Up
CREATE TABLE IF NOT EXISTS user_mfa (
    id BIGINT NOT NULL,
    tenant_id BIGINT NOT NULL DEFAULT 0,
    user_id BIGINT NOT NULL DEFAULT 0,
    totp_secret TEXT,
    totp_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id)
);
CREATE INDEX IF NOT EXISTS idx_user_mfa_user ON user_mfa (user_id);
CREATE INDEX IF NOT EXISTS idx_user_mfa_tenant ON user_mfa (tenant_id, user_id);

COMMENT ON TABLE user_mfa IS '用户 TOTP 二次验证状态';
COMMENT ON COLUMN user_mfa.tenant_id IS '所属租户ID（0=未归属）';
COMMENT ON COLUMN user_mfa.user_id IS '用户ID';
COMMENT ON COLUMN user_mfa.totp_secret IS 'TOTP 密钥（base32，AES-GCM 密文）';
COMMENT ON COLUMN user_mfa.totp_enabled IS '是否启用 TOTP 二次验证';
COMMENT ON COLUMN user_mfa.created_at IS '创建时间';
COMMENT ON COLUMN user_mfa.updated_at IS '更新时间';

-- 密文原样搬迁（两列均为 AES-GCM 密文，格式一致，不重新加解密）。
-- 注意旧 `users.totp_enabled` 在 PG 下是 SMALLINT（见 user 能力迁移 004），
-- 故显式转成 BOOLEAN；id 复用 users.id（一用户至多一条 MFA 记录，主键按表独立），
-- 雪花 ID 全局唯一，不会与后续运行时生成的 user_mfa.id 冲突。
INSERT INTO user_mfa (id, tenant_id, user_id, totp_secret, totp_enabled)
    SELECT id, tenant_id, id, totp_secret, (totp_enabled = 1) FROM users WHERE totp_enabled = 1;

ALTER TABLE users
    DROP COLUMN totp_enabled,
    DROP COLUMN totp_secret;

-- +goose Down
-- 恢复 user 能力 004 迁移的原始列形态（SMALLINT + 默认 0），保证 Down 后可再次 Up。
ALTER TABLE users
    ADD COLUMN totp_secret TEXT,
    ADD COLUMN totp_enabled SMALLINT NOT NULL DEFAULT 0;

UPDATE users u
    SET totp_secret = m.totp_secret,
        totp_enabled = CASE WHEN m.totp_enabled THEN 1 ELSE 0 END
    FROM user_mfa m
    WHERE m.user_id = u.id AND m.totp_enabled = TRUE;

DROP TABLE IF EXISTS user_mfa;
