-- +goose Up
ALTER TABLE capmig_zshared ADD COLUMN tag VARCHAR(64) NULL;
-- +goose Down
ALTER TABLE capmig_zshared DROP COLUMN tag;
