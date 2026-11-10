-- +goose Up
CREATE TABLE IF NOT EXISTS capmig_zitems (
    id BIGINT UNSIGNED NOT NULL,
    title VARCHAR(64) NOT NULL,
    PRIMARY KEY (id)
);
-- +goose Down
DROP TABLE IF EXISTS capmig_zitems;
