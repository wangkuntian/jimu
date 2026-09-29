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
COMMENT ON COLUMN import_jobs.id IS '导入任务ID';
COMMENT ON COLUMN import_jobs.type IS '导入类型（users 等）';
COMMENT ON COLUMN import_jobs.filename IS '文件名';
COMMENT ON COLUMN import_jobs.status IS '状态：pending/processing/completed/failed';
COMMENT ON COLUMN import_jobs.total_rows IS '总行数';
COMMENT ON COLUMN import_jobs.success_rows IS '成功行数';
COMMENT ON COLUMN import_jobs.error_rows IS '失败行数';
COMMENT ON COLUMN import_jobs.errors IS 'JSON 错误详情';
COMMENT ON COLUMN import_jobs.created_by IS '创建人用户ID';
COMMENT ON COLUMN import_jobs.created_at IS '创建时间';
COMMENT ON COLUMN import_jobs.completed_at IS '完成时间';

-- +goose Down
DROP TABLE IF EXISTS import_jobs;
