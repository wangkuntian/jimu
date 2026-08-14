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

-- +goose Down
DROP TABLE IF EXISTS permissions;
