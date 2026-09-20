-- +goose Up
CREATE TABLE IF NOT EXISTS password_histories (
    id BIGINT NOT NULL,
    tenant_id BIGINT NOT NULL DEFAULT 0,
    user_id BIGINT NOT NULL DEFAULT 0,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id)
);
CREATE INDEX IF NOT EXISTS idx_password_histories_user ON password_histories (user_id, id);

COMMENT ON TABLE password_histories IS '密码历史表（防止改回旧密码）';
COMMENT ON COLUMN password_histories.tenant_id IS '所属租户ID（0=未归属）';
COMMENT ON COLUMN password_histories.user_id IS '用户ID';
COMMENT ON COLUMN password_histories.password_hash IS '历史密码哈希（bcrypt）';
COMMENT ON COLUMN password_histories.created_at IS '记录时间';

-- +goose Down
DROP TABLE IF EXISTS password_histories;
