-- +goose Up
CREATE TABLE IF NOT EXISTS login_histories (
    id BIGINT NOT NULL,
    tenant_id BIGINT NOT NULL DEFAULT 0,
    user_id BIGINT NOT NULL DEFAULT 0,
    username VARCHAR(64) NOT NULL,
    status VARCHAR(16) NOT NULL,
    reason VARCHAR(128) NOT NULL DEFAULT '',
    ip VARCHAR(64) NOT NULL DEFAULT '',
    user_agent VARCHAR(256) NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id)
);
CREATE INDEX IF NOT EXISTS idx_login_histories_user ON login_histories (user_id, created_at);
CREATE INDEX IF NOT EXISTS idx_login_histories_tenant ON login_histories (tenant_id, created_at);
CREATE INDEX IF NOT EXISTS idx_login_histories_username ON login_histories (username, created_at);

COMMENT ON TABLE login_histories IS '登录历史表';
COMMENT ON COLUMN login_histories.tenant_id IS '所属租户ID（0=未归属）';
COMMENT ON COLUMN login_histories.user_id IS '用户ID（账号不存在时为0）';
COMMENT ON COLUMN login_histories.username IS '登录名（原样记录，便于排查）';
COMMENT ON COLUMN login_histories.status IS '结果：success/failed/locked';
COMMENT ON COLUMN login_histories.reason IS '失败原因（成功为空）';
COMMENT ON COLUMN login_histories.ip IS '客户端IP';
COMMENT ON COLUMN login_histories.user_agent IS 'User-Agent';
COMMENT ON COLUMN login_histories.created_at IS '发生时间';

-- +goose Down
DROP TABLE IF EXISTS login_histories;
