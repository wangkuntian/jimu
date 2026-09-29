-- +goose Up
CREATE TABLE IF NOT EXISTS capmig_solo_tags (
    id BIGINT NOT NULL,
    label VARCHAR(64) NOT NULL,
    PRIMARY KEY (id)
);
-- +goose Down
DROP TABLE IF EXISTS capmig_solo_tags;
