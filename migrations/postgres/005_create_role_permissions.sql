-- +goose Up
CREATE TABLE IF NOT EXISTS role_permissions (
    role_id BIGINT NOT NULL,
    permission_id BIGINT NOT NULL,
    PRIMARY KEY (role_id, permission_id)
);
CREATE INDEX IF NOT EXISTS idx_role_permissions_permission_id ON role_permissions (permission_id);

COMMENT ON TABLE role_permissions IS '角色-权限关联表';

-- +goose Down
DROP TABLE IF EXISTS role_permissions;
