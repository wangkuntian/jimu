-- +goose Up
ALTER TABLE import_jobs ADD COLUMN IF NOT EXISTS tenant_id BIGINT NOT NULL DEFAULT 0;
CREATE INDEX IF NOT EXISTS idx_import_jobs_tenant_id ON import_jobs (tenant_id);

UPDATE import_jobs SET tenant_id = 1 WHERE tenant_id = 0;

COMMENT ON COLUMN import_jobs.tenant_id IS '所属租户ID（0=未归属）';

-- +goose Down
DROP INDEX IF EXISTS idx_import_jobs_tenant_id;
ALTER TABLE import_jobs DROP COLUMN IF EXISTS tenant_id;
