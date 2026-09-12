-- +goose Up
CREATE TABLE IF NOT EXISTS webauthn_credentials (
    id BIGINT NOT NULL,
    tenant_id BIGINT NOT NULL DEFAULT 0,
    user_id BIGINT NOT NULL DEFAULT 0,
    credential_id VARCHAR(512) NOT NULL,
    public_key BYTEA NOT NULL,
    attestation_type VARCHAR(32) NOT NULL DEFAULT '',
    attestation_format VARCHAR(32) NOT NULL DEFAULT '',
    aaguid VARCHAR(36) NOT NULL DEFAULT '',
    sign_count BIGINT NOT NULL DEFAULT 0,
    transports VARCHAR(255) NOT NULL DEFAULT '',
    backup_eligible BOOLEAN NOT NULL DEFAULT FALSE,
    backup_state BOOLEAN NOT NULL DEFAULT FALSE,
    user_present BOOLEAN NOT NULL DEFAULT FALSE,
    user_verified BOOLEAN NOT NULL DEFAULT FALSE,
    name VARCHAR(64) NOT NULL DEFAULT '',
    last_used_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id)
);
CREATE UNIQUE INDEX IF NOT EXISTS uk_webauthn_credentials_credential ON webauthn_credentials (credential_id);
CREATE INDEX IF NOT EXISTS idx_webauthn_credentials_user ON webauthn_credentials (user_id);

COMMENT ON TABLE webauthn_credentials IS 'WebAuthn/通行密钥凭证表（无密码登录与二次验证）';
COMMENT ON COLUMN webauthn_credentials.tenant_id IS '所属租户ID（0=未归属）';
COMMENT ON COLUMN webauthn_credentials.user_id IS '用户ID';
COMMENT ON COLUMN webauthn_credentials.credential_id IS '凭证ID（authenticator 生成，base64url 编码存储）';
COMMENT ON COLUMN webauthn_credentials.public_key IS '凭证公钥（COSE 编码原始字节，登录验签用）';
COMMENT ON COLUMN webauthn_credentials.attestation_type IS '认证器证明类型（none/basic_full 等）';
COMMENT ON COLUMN webauthn_credentials.attestation_format IS '证明语句格式（none/packed 等）';
COMMENT ON COLUMN webauthn_credentials.aaguid IS '认证器 AAGUID（十六进制，可空）';
COMMENT ON COLUMN webauthn_credentials.sign_count IS '签名计数器（克隆检测用）';
COMMENT ON COLUMN webauthn_credentials.transports IS '支持的传输方式（JSON 数组）';
COMMENT ON COLUMN webauthn_credentials.backup_eligible IS '是否可备份（多设备同步的通行密钥）';
COMMENT ON COLUMN webauthn_credentials.backup_state IS '当前是否已被备份';
COMMENT ON COLUMN webauthn_credentials.user_present IS '注册时是否校验用户在场';
COMMENT ON COLUMN webauthn_credentials.user_verified IS '注册时是否校验用户身份（生物识别/PIN）';
COMMENT ON COLUMN webauthn_credentials.name IS '凭证名称（用户自定义，便于识别）';
COMMENT ON COLUMN webauthn_credentials.last_used_at IS '最近登录时间';
COMMENT ON COLUMN webauthn_credentials.created_at IS '创建时间';
COMMENT ON COLUMN webauthn_credentials.updated_at IS '更新时间';

-- +goose Down
DROP TABLE IF EXISTS webauthn_credentials;
