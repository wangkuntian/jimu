-- +goose Up
CREATE TABLE IF NOT EXISTS capmig_users (
    id BIGSERIAL PRIMARY KEY,
    username VARCHAR(64) NOT NULL
);
-- +goose Down
DROP TABLE IF EXISTS capmig_users;
