-- +goose Up
CREATE TABLE IF NOT EXISTS audit_logs (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '审计日志ID',
    user_id BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '操作用户ID',
    username VARCHAR(64) NOT NULL DEFAULT '' COMMENT '操作用户名',
    action VARCHAR(64) NOT NULL COMMENT '操作类型（如 CREATE/UPDATE/DELETE）',
    resource VARCHAR(128) NOT NULL DEFAULT '' COMMENT '操作资源',
    detail TEXT COMMENT '操作详情（JSON）',
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

-- +goose Down
DROP TABLE IF EXISTS audit_logs;
