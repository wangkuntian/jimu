-- +goose Up
CREATE TABLE IF NOT EXISTS capmig_zshared (
    id BIGINT UNSIGNED NOT NULL,
    ref BIGINT UNSIGNED NOT NULL,
    PRIMARY KEY (id)
);
-- +goose Down
DROP TABLE IF EXISTS capmig_zshared;
