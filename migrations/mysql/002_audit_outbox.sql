-- +goose Up
CREATE TABLE IF NOT EXISTS audit_logs (
    id BIGINT UNSIGNED NOT NULL COMMENT '审计日志ID',
    user_id BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '操作用户ID',
    username VARCHAR(64) NOT NULL DEFAULT '' COMMENT '操作用户名',
    action VARCHAR(64) NOT NULL COMMENT '操作类型（如 CREATE/UPDATE/DELETE）',
    resource VARCHAR(128) NOT NULL DEFAULT '' COMMENT '操作资源',
    detail TEXT COMMENT '操作详情（JSON）',
    changes TEXT COMMENT '字段变更记录（JSON数组）',
    ip VARCHAR(64) NOT NULL DEFAULT '' COMMENT '客户端IP',
    method VARCHAR(16) NOT NULL DEFAULT '' COMMENT 'HTTP方法',
    path VARCHAR(256) NOT NULL DEFAULT '' COMMENT '请求路径',
    status INT NOT NULL DEFAULT 0 COMMENT 'HTTP状态码',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '操作时间',
    PRIMARY KEY (id),
    INDEX idx_user_id (user_id),
    INDEX idx_action (action),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='审计日志表';

CREATE TABLE IF NOT EXISTS outbox_events (
    id BIGINT UNSIGNED NOT NULL COMMENT '事件ID',
    aggregate_id VARCHAR(128) NOT NULL COMMENT '聚合根ID',
    event_type VARCHAR(128) NOT NULL COMMENT '事件类型',
    payload JSON NOT NULL COMMENT '事件数据',
    metadata JSON NULL COMMENT '元数据',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    published_at TIMESTAMP NULL DEFAULT NULL COMMENT '发布时间（NULL=未发布）',
    retry_count INT NOT NULL DEFAULT 0 COMMENT '重试次数',
    PRIMARY KEY (id),
    INDEX idx_outbox_aggregate_id (aggregate_id),
    INDEX idx_outbox_event_type (event_type),
    INDEX idx_outbox_published_at (published_at),
    INDEX idx_outbox_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Outbox事件表';

-- +goose Down
DROP TABLE IF EXISTS outbox_events;
DROP TABLE IF EXISTS audit_logs;
