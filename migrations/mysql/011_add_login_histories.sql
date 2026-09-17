-- +goose Up
CREATE TABLE IF NOT EXISTS login_histories (
    id BIGINT UNSIGNED NOT NULL COMMENT '记录ID',
    tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '所属租户ID（0=未归属）',
    user_id BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '用户ID（账号不存在时为0）',
    username VARCHAR(64) NOT NULL COMMENT '登录名（原样记录，便于排查）',
    status VARCHAR(16) NOT NULL COMMENT '结果：success/failed/locked',
    reason VARCHAR(128) NOT NULL DEFAULT '' COMMENT '失败原因（成功为空）',
    ip VARCHAR(64) NOT NULL DEFAULT '' COMMENT '客户端IP',
    user_agent VARCHAR(256) NOT NULL DEFAULT '' COMMENT 'User-Agent',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '发生时间',
    PRIMARY KEY (id),
    INDEX idx_login_histories_user (user_id, created_at),
    INDEX idx_login_histories_tenant (tenant_id, created_at),
    INDEX idx_login_histories_username (username, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='登录历史表';

-- +goose Down
DROP TABLE IF EXISTS login_histories;
