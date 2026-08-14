-- +goose Up
CREATE TABLE IF NOT EXISTS audit_logs (
    id BIGINT NOT NULL,
    user_id BIGINT NOT NULL DEFAULT 0,
    username VARCHAR(64) NOT NULL DEFAULT '',
    action VARCHAR(64) NOT NULL,
    resource VARCHAR(128) NOT NULL DEFAULT '',
    detail TEXT,
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

-- +goose Down
DROP TABLE IF EXISTS audit_logs;
