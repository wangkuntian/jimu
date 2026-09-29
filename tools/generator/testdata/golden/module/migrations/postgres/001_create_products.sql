-- +goose Up
-- ponytail: PostgreSQL 模板从 MySQL 直译（SERIAL 大致等价 AUTO_INCREMENT），未覆盖
-- 方言细节（部分索引、IDENTITY 语义等），需要时生成者手改本文件。
CREATE TABLE IF NOT EXISTS products (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    description VARCHAR(255) DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL DEFAULT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS uk_products_name ON products (name);
CREATE INDEX IF NOT EXISTS idx_products_deleted_at ON products (deleted_at);

-- +goose Down
DROP TABLE IF EXISTS products;
