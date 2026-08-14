-- +goose Up
CREATE TABLE user_oauth_bindings (
    id BIGINT UNSIGNED NOT NULL COMMENT '主键',
    user_id BIGINT UNSIGNED NOT NULL COMMENT '用户 ID',
    provider VARCHAR(32) NOT NULL COMMENT '提供商（google/github/wechat）',
    subject VARCHAR(128) NOT NULL COMMENT '提供商内唯一 ID',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    deleted_at DATETIME DEFAULT NULL COMMENT '软删除时间',
    PRIMARY KEY (id),
    UNIQUE KEY uk_oauth_binding (provider, subject),
    KEY idx_oauth_user_id (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='第三方登录绑定表';

-- +goose Down
DROP TABLE user_oauth_bindings;
