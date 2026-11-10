-- +goose Up
CREATE TABLE IF NOT EXISTS capmig_solo_items (
    id BIGINT NOT NULL,
    name VARCHAR(64) NOT NULL,
    PRIMARY KEY (id)
);
-- +goose Down
DROP TABLE IF EXISTS capmig_solo_items;
