-- +goose Up
CREATE TABLE IF NOT EXISTS webauthn_credentials (
    id BIGINT UNSIGNED NOT NULL COMMENT '凭证ID',
    tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '所属租户ID（0=未归属）',
    user_id BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '用户ID',
    credential_id VARCHAR(512) NOT NULL COMMENT '凭证ID（authenticator 生成，base64url 编码存储）',
    public_key BLOB NOT NULL COMMENT '凭证公钥（COSE 编码原始字节，登录验签用）',
    attestation_type VARCHAR(32) NOT NULL DEFAULT '' COMMENT '认证器证明类型（none/basic_full 等）',
    attestation_format VARCHAR(32) NOT NULL DEFAULT '' COMMENT '证明语句格式（none/packed 等）',
    aaguid VARCHAR(36) NOT NULL DEFAULT '' COMMENT '认证器 AAGUID（十六进制，可空）',
    sign_count BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '签名计数器（克隆检测用）',
    transports VARCHAR(255) NOT NULL DEFAULT '' COMMENT '支持的传输方式（JSON 数组）',
    backup_eligible TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否可备份（多设备同步的通行密钥）',
    backup_state TINYINT(1) NOT NULL DEFAULT 0 COMMENT '当前是否已被备份',
    user_present TINYINT(1) NOT NULL DEFAULT 0 COMMENT '注册时是否校验用户在场',
    user_verified TINYINT(1) NOT NULL DEFAULT 0 COMMENT '注册时是否校验用户身份（生物识别/PIN）',
    name VARCHAR(64) NOT NULL DEFAULT '' COMMENT '凭证名称（用户自定义，便于识别）',
    last_used_at TIMESTAMP NULL COMMENT '最近登录时间',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (id),
    UNIQUE INDEX uk_webauthn_credentials_credential (credential_id),
    INDEX idx_webauthn_credentials_user (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='WebAuthn/通行密钥凭证表（无密码登录与二次验证）';

-- +goose Down
DROP TABLE IF EXISTS webauthn_credentials;
