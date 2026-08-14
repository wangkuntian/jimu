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

-- +goose Down
DROP TABLE IF EXISTS roles;
