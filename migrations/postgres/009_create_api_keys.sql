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

-- +goose Down
DROP TABLE IF EXISTS api_keys;
