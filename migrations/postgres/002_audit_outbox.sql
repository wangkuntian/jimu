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

CREATE TABLE IF NOT EXISTS outbox_events (
    id BIGINT NOT NULL,
    aggregate_id VARCHAR(128) NOT NULL,
    event_type VARCHAR(128) NOT NULL,
    payload JSON NOT NULL,
    metadata JSON,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    published_at TIMESTAMP,
    retry_count INT NOT NULL DEFAULT 0,
    PRIMARY KEY (id)
);
CREATE INDEX IF NOT EXISTS idx_outbox_aggregate_id ON outbox_events (aggregate_id);
CREATE INDEX IF NOT EXISTS idx_outbox_event_type ON outbox_events (event_type);
CREATE INDEX IF NOT EXISTS idx_outbox_published_at ON outbox_events (published_at);
CREATE INDEX IF NOT EXISTS idx_outbox_created_at ON outbox_events (created_at);

COMMENT ON TABLE outbox_events IS 'Outbox事件表';
COMMENT ON COLUMN outbox_events.id IS '事件ID';
COMMENT ON COLUMN outbox_events.aggregate_id IS '聚合根ID';
COMMENT ON COLUMN outbox_events.event_type IS '事件类型';
COMMENT ON COLUMN outbox_events.payload IS '事件数据';
COMMENT ON COLUMN outbox_events.metadata IS '元数据';
COMMENT ON COLUMN outbox_events.created_at IS '创建时间';
COMMENT ON COLUMN outbox_events.published_at IS '发布时间（NULL=未发布）';
COMMENT ON COLUMN outbox_events.retry_count IS '重试次数';

-- +goose Down
DROP TABLE IF EXISTS outbox_events;
DROP TABLE IF EXISTS audit_logs;
