-- +goose Up
ALTER TABLE import_jobs ADD COLUMN tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '所属租户ID（0=未归属）' AFTER type;
CREATE INDEX idx_import_jobs_tenant_id ON import_jobs (tenant_id);

UPDATE import_jobs SET tenant_id = 1 WHERE tenant_id = 0;

-- +goose Down
DROP INDEX idx_import_jobs_tenant_id ON import_jobs;
ALTER TABLE import_jobs DROP COLUMN tenant_id;
