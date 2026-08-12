-- +goose Up
CREATE TABLE IF NOT EXISTS import_jobs (
    id BIGINT UNSIGNED NOT NULL COMMENT '导入任务ID',
    type VARCHAR(64) NOT NULL COMMENT '导入类型（users 等）',
    filename VARCHAR(255) NOT NULL COMMENT '文件名',
    status VARCHAR(16) NOT NULL DEFAULT 'pending' COMMENT '状态：pending/processing/completed/failed',
    total_rows INT NOT NULL DEFAULT 0 COMMENT '总行数',
    success_rows INT NOT NULL DEFAULT 0 COMMENT '成功行数',
    error_rows INT NOT NULL DEFAULT 0 COMMENT '失败行数',
    errors TEXT COMMENT 'JSON 错误详情',
    created_by BIGINT UNSIGNED COMMENT '创建人用户ID',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    completed_at TIMESTAMP NULL COMMENT '完成时间',
    PRIMARY KEY (id),
    INDEX idx_status (status),
    INDEX idx_created_by (created_by)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='数据导入任务表';

-- +goose Down
DROP TABLE IF EXISTS import_jobs;
