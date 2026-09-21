-- +goose Up
CREATE TABLE IF NOT EXISTS permissions (
    id BIGINT NOT NULL,
    name VARCHAR(128) NOT NULL,
    resource VARCHAR(64) NOT NULL,
    action VARCHAR(32) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    PRIMARY KEY (id),
    CONSTRAINT uk_permissions_resource_action UNIQUE (resource, action)
);
CREATE INDEX IF NOT EXISTS idx_permissions_deleted_at ON permissions (deleted_at);

COMMENT ON TABLE permissions IS '权限表';
COMMENT ON COLUMN permissions.id IS '权限ID';
COMMENT ON COLUMN permissions.name IS '权限名称';
COMMENT ON COLUMN permissions.resource IS '资源路径（如 /api/v1/users）';
COMMENT ON COLUMN permissions.action IS '操作类型（GET/POST/PUT/DELETE）';
COMMENT ON COLUMN permissions.created_at IS '创建时间';
COMMENT ON COLUMN permissions.updated_at IS '更新时间';
COMMENT ON COLUMN permissions.deleted_at IS '删除时间（软删除）';

CREATE TABLE IF NOT EXISTS role_permissions (
    role_id BIGINT NOT NULL,
    permission_id BIGINT NOT NULL,
    PRIMARY KEY (role_id, permission_id)
);
CREATE INDEX IF NOT EXISTS idx_role_permissions_permission_id ON role_permissions (permission_id);

COMMENT ON TABLE role_permissions IS '角色-权限关联表';
COMMENT ON COLUMN role_permissions.role_id IS '角色ID';
COMMENT ON COLUMN role_permissions.permission_id IS '权限ID';

-- +goose Down
DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS permissions;
