-- +goose Up
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS tenant_id BIGINT NOT NULL DEFAULT 0;
CREATE INDEX IF NOT EXISTS idx_jobs_tenant_id ON jobs (tenant_id);

ALTER TABLE job_history ADD COLUMN IF NOT EXISTS tenant_id BIGINT NOT NULL DEFAULT 0;
CREATE INDEX IF NOT EXISTS idx_job_history_tenant_id ON job_history (tenant_id);

ALTER TABLE dead_letters ADD COLUMN IF NOT EXISTS tenant_id BIGINT NOT NULL DEFAULT 0;
CREATE INDEX IF NOT EXISTS idx_dead_letters_tenant_id ON dead_letters (tenant_id);

ALTER TABLE import_jobs ADD COLUMN IF NOT EXISTS tenant_id BIGINT NOT NULL DEFAULT 0;
CREATE INDEX IF NOT EXISTS idx_import_jobs_tenant_id ON import_jobs (tenant_id);

-- 存量数据迁入默认租户（id=1，与 platform/tenant.DefaultTenantID 保持一致）
UPDATE jobs SET tenant_id = 1 WHERE tenant_id = 0;
UPDATE job_history SET tenant_id = 1 WHERE tenant_id = 0;
UPDATE dead_letters SET tenant_id = 1 WHERE tenant_id = 0;
UPDATE import_jobs SET tenant_id = 1 WHERE tenant_id = 0;

COMMENT ON COLUMN jobs.tenant_id IS '所属租户ID（0=未归属）';
COMMENT ON COLUMN job_history.tenant_id IS '所属租户ID（0=未归属）';
COMMENT ON COLUMN dead_letters.tenant_id IS '所属租户ID（0=未归属）';
COMMENT ON COLUMN import_jobs.tenant_id IS '所属租户ID（0=未归属）';

-- +goose Down
DROP INDEX IF EXISTS idx_import_jobs_tenant_id;
ALTER TABLE import_jobs DROP COLUMN IF EXISTS tenant_id;
DROP INDEX IF EXISTS idx_dead_letters_tenant_id;
ALTER TABLE dead_letters DROP COLUMN IF EXISTS tenant_id;
DROP INDEX IF EXISTS idx_job_history_tenant_id;
ALTER TABLE job_history DROP COLUMN IF EXISTS tenant_id;
DROP INDEX IF EXISTS idx_jobs_tenant_id;
ALTER TABLE jobs DROP COLUMN IF EXISTS tenant_id;
