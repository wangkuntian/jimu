-- +goose Up
CREATE TABLE IF NOT EXISTS users (
    id BIGINT NOT NULL,
    username VARCHAR(64) NOT NULL UNIQUE,
    password VARCHAR(255) NOT NULL,
    status SMALLINT NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    PRIMARY KEY (id)
);
CREATE INDEX IF NOT EXISTS idx_username ON users (username);
CREATE INDEX IF NOT EXISTS idx_deleted_at ON users (deleted_at);

COMMENT ON TABLE users IS '用户表';

-- +goose Down
DROP TABLE IF EXISTS users;
