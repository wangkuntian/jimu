-- +goose Up
CREATE TABLE IF NOT EXISTS users (
    id BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
    username VARCHAR(64) NOT NULL UNIQUE COMMENT '用户名',
    email TEXT NULL COMMENT '邮箱（AES-GCM 密文）',
    email_hash VARCHAR(64) NULL COMMENT '邮箱盲索引（HMAC-SHA256，精确查询与唯一约束）',
    phone VARCHAR(255) NULL COMMENT '手机号（AES-GCM 密文）',
    phone_hash VARCHAR(64) NULL COMMENT '手机号盲索引（HMAC-SHA256，精确查询与唯一约束）',
    password VARCHAR(255) NOT NULL COMMENT '密码（bcrypt哈希）',
    status TINYINT NOT NULL DEFAULT 1 COMMENT '状态：1-启用 0-禁用',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    deleted_at TIMESTAMP NULL DEFAULT NULL COMMENT '删除时间（软删除）',
    PRIMARY KEY (id),
    UNIQUE INDEX idx_users_email_hash (email_hash),
    UNIQUE INDEX idx_users_phone_hash (phone_hash),
    INDEX idx_username (username),
    INDEX idx_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户表';

-- +goose Down
DROP TABLE IF EXISTS users;
