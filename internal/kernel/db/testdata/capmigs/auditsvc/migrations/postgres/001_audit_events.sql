-- +goose Up
CREATE TABLE IF NOT EXISTS capmig_audit_events (
    id BIGSERIAL PRIMARY KEY,
    message VARCHAR(255) NOT NULL
);
-- +goose Down
DROP TABLE IF EXISTS capmig_audit_events;
