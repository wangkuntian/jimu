-- +goose Up
ALTER TABLE jobs ADD COLUMN tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '所属租户ID（0=未归属）' AFTER id;
CREATE INDEX idx_jobs_tenant_id ON jobs (tenant_id);

ALTER TABLE job_history ADD COLUMN tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '所属租户ID（0=未归属）' AFTER job_id;
CREATE INDEX idx_job_history_tenant_id ON job_history (tenant_id);

ALTER TABLE dead_letters ADD COLUMN tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '所属租户ID（0=未归属）' AFTER job_id;
CREATE INDEX idx_dead_letters_tenant_id ON dead_letters (tenant_id);

-- 存量数据迁入默认租户（id=1，与 kernel/tenant.DefaultTenantID 保持一致）
UPDATE jobs SET tenant_id = 1 WHERE tenant_id = 0;
UPDATE job_history SET tenant_id = 1 WHERE tenant_id = 0;
UPDATE dead_letters SET tenant_id = 1 WHERE tenant_id = 0;

-- +goose Down
DROP INDEX idx_dead_letters_tenant_id ON dead_letters;
ALTER TABLE dead_letters DROP COLUMN tenant_id;
DROP INDEX idx_job_history_tenant_id ON job_history;
ALTER TABLE job_history DROP COLUMN tenant_id;
DROP INDEX idx_jobs_tenant_id ON jobs;
ALTER TABLE jobs DROP COLUMN tenant_id;
