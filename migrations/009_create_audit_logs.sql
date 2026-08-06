-- +goose Up
CREATE TABLE IF NOT EXISTS audit_logs (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '审计日志 ID',
    admin_id BIGINT UNSIGNED NOT NULL COMMENT '管理员用户 ID',
    admin_name VARCHAR(64) COMMENT '管理员用户名',
    action VARCHAR(64) NOT NULL COMMENT '操作类型（如 user.create）',
    resource VARCHAR(128) NOT NULL COMMENT '操作资源（如 user:123）',
    detail TEXT COMMENT '变更详情（JSON）',
    ip VARCHAR(64) COMMENT '客户端 IP',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '操作时间',
    PRIMARY KEY (id),
    INDEX idx_admin_id (admin_id),
    INDEX idx_action (action),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='审计日志表';

-- +goose Down
DROP TABLE IF EXISTS audit_logs;
