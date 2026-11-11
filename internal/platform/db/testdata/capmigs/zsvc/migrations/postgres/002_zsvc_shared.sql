-- +goose Up
CREATE TABLE IF NOT EXISTS capmig_zshared (
    id BIGSERIAL PRIMARY KEY,
    ref BIGINT NOT NULL
);
-- +goose Down
DROP TABLE IF EXISTS capmig_zshared;
