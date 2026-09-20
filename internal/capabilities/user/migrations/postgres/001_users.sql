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

-- +goose Down
DROP TABLE IF EXISTS users;
