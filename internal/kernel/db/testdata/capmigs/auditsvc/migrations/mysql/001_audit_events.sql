-- +goose Up
CREATE TABLE IF NOT EXISTS capmig_audit_events (
    id BIGINT UNSIGNED NOT NULL,
    message VARCHAR(255) NOT NULL,
    PRIMARY KEY (id)
);
-- +goose Down
DROP TABLE IF EXISTS capmig_audit_events;
