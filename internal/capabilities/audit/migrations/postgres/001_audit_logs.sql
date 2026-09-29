-- +goose Up
CREATE TABLE IF NOT EXISTS audit_logs (
    id BIGINT NOT NULL,
    user_id BIGINT NOT NULL DEFAULT 0,
    username VARCHAR(64) NOT NULL DEFAULT '',
    action VARCHAR(64) NOT NULL,
    resource VARCHAR(128) NOT NULL DEFAULT '',
    detail TEXT,
    changes TEXT,
    ip VARCHAR(64) NOT NULL DEFAULT '',
    method VARCHAR(16) NOT NULL DEFAULT '',
    path VARCHAR(256) NOT NULL DEFAULT '',
    status INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id)
);
CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON audit_logs (user_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs (action);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs (created_at);

COMMENT ON TABLE audit_logs IS '审计日志表';
COMMENT ON COLUMN audit_logs.id IS '审计日志ID';
COMMENT ON COLUMN audit_logs.user_id IS '操作用户ID';
COMMENT ON COLUMN audit_logs.username IS '操作用户名';
COMMENT ON COLUMN audit_logs.action IS '操作类型（如 CREATE/UPDATE/DELETE）';
COMMENT ON COLUMN audit_logs.resource IS '操作资源';
COMMENT ON COLUMN audit_logs.detail IS '操作详情（JSON）';
COMMENT ON COLUMN audit_logs.changes IS '字段变更记录（JSON数组）';
COMMENT ON COLUMN audit_logs.ip IS '客户端IP';
COMMENT ON COLUMN audit_logs.method IS 'HTTP方法';
COMMENT ON COLUMN audit_logs.path IS '请求路径';
COMMENT ON COLUMN audit_logs.status IS 'HTTP状态码';
COMMENT ON COLUMN audit_logs.created_at IS '操作时间';

-- +goose Down
DROP TABLE IF EXISTS audit_logs;
