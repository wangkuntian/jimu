-- +goose Up
CREATE TABLE IF NOT EXISTS users (
    id BIGINT NOT NULL,
    username VARCHAR(64) NOT NULL UNIQUE,
    email TEXT,
    email_hash VARCHAR(64),
    phone VARCHAR(255),
    phone_hash VARCHAR(64),
    password VARCHAR(255) NOT NULL,
    status SMALLINT NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    PRIMARY KEY (id)
);
CREATE INDEX IF NOT EXISTS idx_username ON users (username);
CREATE INDEX IF NOT EXISTS idx_deleted_at ON users (deleted_at);
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_hash ON users (email_hash);
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_phone_hash ON users (phone_hash);

COMMENT ON TABLE users IS '用户表';
COMMENT ON COLUMN users.id IS '用户ID';
COMMENT ON COLUMN users.username IS '用户名';
COMMENT ON COLUMN users.email IS '邮箱（AES-GCM 密文）';
COMMENT ON COLUMN users.email_hash IS '邮箱盲索引（HMAC-SHA256，精确查询与唯一约束）';
COMMENT ON COLUMN users.phone IS '手机号（AES-GCM 密文）';
COMMENT ON COLUMN users.phone_hash IS '手机号盲索引（HMAC-SHA256，精确查询与唯一约束）';
COMMENT ON COLUMN users.password IS '密码（bcrypt哈希）';
COMMENT ON COLUMN users.status IS '状态：1-启用 0-禁用';
COMMENT ON COLUMN users.created_at IS '创建时间';
COMMENT ON COLUMN users.updated_at IS '更新时间';
COMMENT ON COLUMN users.deleted_at IS '删除时间（软删除）';

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

CREATE TABLE IF NOT EXISTS user_roles (
    user_id BIGINT NOT NULL,
    role_id BIGINT NOT NULL,
    PRIMARY KEY (user_id, role_id)
);
CREATE INDEX IF NOT EXISTS idx_user_roles_role_id ON user_roles (role_id);

COMMENT ON TABLE user_roles IS '用户-角色关联表';
COMMENT ON COLUMN user_roles.user_id IS '用户ID';
COMMENT ON COLUMN user_roles.role_id IS '角色ID';

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
DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS permissions;
DROP TABLE IF EXISTS roles;
DROP TABLE IF EXISTS users;
