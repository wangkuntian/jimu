-- +goose Up
CREATE TABLE IF NOT EXISTS capmig_users (
    id BIGINT UNSIGNED NOT NULL,
    username VARCHAR(64) NOT NULL,
    PRIMARY KEY (id)
);
-- +goose Down
DROP TABLE IF EXISTS capmig_users;
