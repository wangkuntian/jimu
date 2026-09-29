-- +goose Up
ALTER TABLE capmig_users ADD COLUMN note VARCHAR(128);
-- +goose Down
ALTER TABLE capmig_users DROP COLUMN note;
