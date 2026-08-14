-- +goose Up
CREATE TABLE IF NOT EXISTS user_roles (
    user_id BIGINT NOT NULL,
    role_id BIGINT NOT NULL,
    PRIMARY KEY (user_id, role_id)
);
CREATE INDEX IF NOT EXISTS idx_user_roles_role_id ON user_roles (role_id);

COMMENT ON TABLE user_roles IS '用户-角色关联表';

-- +goose Down
DROP TABLE IF EXISTS user_roles;
