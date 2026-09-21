-- +goose Up
CREATE TABLE IF NOT EXISTS roles (
    id BIGINT NOT NULL,
    name VARCHAR(64) NOT NULL UNIQUE,
    description VARCHAR(255) DEFAULT '',
    status SMALLINT NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    PRIMARY KEY (id)
);
CREATE INDEX IF NOT EXISTS idx_roles_deleted_at ON roles (deleted_at);

COMMENT ON TABLE roles IS '角色表';
COMMENT ON COLUMN roles.id IS '角色ID';
COMMENT ON COLUMN roles.name IS '角色名称';
COMMENT ON COLUMN roles.description IS '角色描述';
COMMENT ON COLUMN roles.status IS '状态：1-启用 0-禁用';
COMMENT ON COLUMN roles.created_at IS '创建时间';
COMMENT ON COLUMN roles.updated_at IS '更新时间';
COMMENT ON COLUMN roles.deleted_at IS '删除时间（软删除）';

CREATE TABLE IF NOT EXISTS user_roles (
    user_id BIGINT NOT NULL,
    role_id BIGINT NOT NULL,
    PRIMARY KEY (user_id, role_id)
);
CREATE INDEX IF NOT EXISTS idx_user_roles_role_id ON user_roles (role_id);

COMMENT ON TABLE user_roles IS '用户-角色关联表';
COMMENT ON COLUMN user_roles.user_id IS '用户ID';
COMMENT ON COLUMN user_roles.role_id IS '角色ID';

-- +goose Down
DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS roles;
