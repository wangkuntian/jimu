-- +goose Up
CREATE TABLE IF NOT EXISTS permissions (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '权限ID',
    name VARCHAR(128) NOT NULL COMMENT '权限名称',
    resource VARCHAR(64) NOT NULL COMMENT '资源路径（如 /api/v1/users）',
    action VARCHAR(32) NOT NULL COMMENT '操作类型（GET/POST/PUT/DELETE）',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    deleted_at TIMESTAMP NULL DEFAULT NULL COMMENT '删除时间（软删除）',
    PRIMARY KEY (id),
    UNIQUE KEY uk_resource_action (resource, action),
    INDEX idx_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='权限表';

-- +goose Down
DROP TABLE IF EXISTS permissions;
