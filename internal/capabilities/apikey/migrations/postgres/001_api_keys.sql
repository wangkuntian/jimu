-- +goose Up
CREATE TABLE IF NOT EXISTS api_keys (
    id BIGINT NOT NULL,
    name VARCHAR(64) NOT NULL,
    key_prefix VARCHAR(16) NOT NULL,
    key_hash VARCHAR(64) NOT NULL,
    scopes TEXT,
    enabled SMALLINT NOT NULL DEFAULT 1,
    expires_at TIMESTAMP,
    last_used TIMESTAMP,
    use_count BIGINT NOT NULL DEFAULT 0,
    created_by BIGINT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id)
);
CREATE INDEX IF NOT EXISTS idx_api_keys_key_hash ON api_keys (key_hash);
CREATE INDEX IF NOT EXISTS idx_api_keys_created_by ON api_keys (created_by);

COMMENT ON TABLE api_keys IS 'API 密钥表';
COMMENT ON COLUMN api_keys.id IS 'API Key ID';
COMMENT ON COLUMN api_keys.name IS 'Key 名称';
COMMENT ON COLUMN api_keys.key_prefix IS 'Key 前缀（用于识别）';
COMMENT ON COLUMN api_keys.key_hash IS 'SHA-256 哈希';
COMMENT ON COLUMN api_keys.scopes IS '权限范围（JSON 数组）';
COMMENT ON COLUMN api_keys.enabled IS '是否启用';
COMMENT ON COLUMN api_keys.expires_at IS '过期时间';
COMMENT ON COLUMN api_keys.last_used IS '最后使用时间';
COMMENT ON COLUMN api_keys.use_count IS '使用次数';
COMMENT ON COLUMN api_keys.created_by IS '创建者用户 ID';
COMMENT ON COLUMN api_keys.created_at IS '创建时间';

-- +goose Down
DROP TABLE IF EXISTS api_keys;
