-- +goose Up
CREATE TABLE IF NOT EXISTS import_jobs (
    id BIGINT NOT NULL,
    type VARCHAR(64) NOT NULL,
    filename VARCHAR(255) NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'pending',
    total_rows INT NOT NULL DEFAULT 0,
    success_rows INT NOT NULL DEFAULT 0,
    error_rows INT NOT NULL DEFAULT 0,
    errors TEXT,
    created_by BIGINT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    PRIMARY KEY (id)
);
CREATE INDEX IF NOT EXISTS idx_import_jobs_status ON import_jobs (status);
CREATE INDEX IF NOT EXISTS idx_import_jobs_created_by ON import_jobs (created_by);

COMMENT ON TABLE import_jobs IS '数据导入任务表';

-- +goose Down
DROP TABLE IF EXISTS import_jobs;
