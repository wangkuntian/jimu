-- +goose Up
CREATE TABLE IF NOT EXISTS capmig_zitems (
    id BIGSERIAL PRIMARY KEY,
    title VARCHAR(64) NOT NULL
);
-- +goose Down
DROP TABLE IF EXISTS capmig_zitems;
