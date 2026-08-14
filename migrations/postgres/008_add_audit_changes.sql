-- +goose Up
ALTER TABLE audit_logs ADD COLUMN changes TEXT;

COMMENT ON COLUMN audit_logs.changes IS '字段变更记录（JSON数组）';

-- +goose Down
ALTER TABLE audit_logs DROP COLUMN changes;
