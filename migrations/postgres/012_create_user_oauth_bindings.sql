-- +goose Up
CREATE TABLE IF NOT EXISTS user_oauth_bindings (
    id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    provider VARCHAR(32) NOT NULL,
    subject VARCHAR(128) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    PRIMARY KEY (id),
    CONSTRAINT uk_oauth_binding UNIQUE (provider, subject)
);
CREATE INDEX IF NOT EXISTS idx_oauth_user_id ON user_oauth_bindings (user_id);

COMMENT ON TABLE user_oauth_bindings IS '第三方登录绑定表';

-- +goose Down
DROP TABLE IF EXISTS user_oauth_bindings;
