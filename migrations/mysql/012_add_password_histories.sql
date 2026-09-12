-- +goose Up
CREATE TABLE IF NOT EXISTS password_histories (
    id BIGINT UNSIGNED NOT NULL COMMENT '记录ID',
    tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '所属租户ID（0=未归属）',
    user_id BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '用户ID',
    password_hash VARCHAR(255) NOT NULL COMMENT '历史密码哈希（bcrypt）',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '记录时间',
    PRIMARY KEY (id),
    INDEX idx_password_histories_user (user_id, id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='密码历史表（防止改回旧密码）';

-- +goose Down
DROP TABLE IF EXISTS password_histories;
