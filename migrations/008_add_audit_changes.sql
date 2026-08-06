-- +goose Up
ALTER TABLE audit_logs ADD COLUMN changes TEXT COMMENT '字段变更记录（JSON数组）' AFTER detail;

-- +goose Down
ALTER TABLE audit_logs DROP COLUMN changes;
