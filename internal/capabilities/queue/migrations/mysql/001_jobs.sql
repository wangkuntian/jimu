-- +goose Up
CREATE TABLE IF NOT EXISTS jobs (
    id BIGINT UNSIGNED NOT NULL COMMENT 'Job ID',
    type VARCHAR(64) NOT NULL COMMENT 'Job 类型（send_email 等）',
    payload TEXT COMMENT 'JSON 载荷',
    status VARCHAR(16) NOT NULL DEFAULT 'pending' COMMENT '状态：pending/running/success/failed/dead',
    priority INT NOT NULL DEFAULT 5 COMMENT '优先级：0-9，越小越优先',
    attempts INT NOT NULL DEFAULT 0 COMMENT '已尝试次数',
    max_attempts INT NOT NULL DEFAULT 3 COMMENT '最大重试次数',
    next_run_at TIMESTAMP NULL COMMENT '下次执行时间',
    error TEXT COMMENT '最近错误信息',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (id),
    INDEX idx_status_next_run (status, next_run_at),
    INDEX idx_type (type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Job 队列表';

CREATE TABLE IF NOT EXISTS job_history (
    id BIGINT UNSIGNED NOT NULL COMMENT '历史记录 ID',
    job_id BIGINT UNSIGNED NOT NULL COMMENT 'Job ID',
    status VARCHAR(16) NOT NULL COMMENT '执行结果：success/failed',
    error TEXT COMMENT '错误信息',
    duration_ms BIGINT COMMENT '执行耗时（毫秒）',
    started_at TIMESTAMP NULL COMMENT '开始时间',
    ended_at TIMESTAMP NULL COMMENT '结束时间',
    PRIMARY KEY (id),
    INDEX idx_job_id (job_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Job 执行历史表';

CREATE TABLE IF NOT EXISTS dead_letters (
    id BIGINT UNSIGNED NOT NULL COMMENT '死信 ID',
    job_id BIGINT UNSIGNED NOT NULL COMMENT '原 Job ID',
    type VARCHAR(64) NOT NULL COMMENT 'Job 类型',
    payload TEXT COMMENT 'JSON 载荷',
    fail_reason TEXT COMMENT '失败原因',
    failed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '失败时间',
    resolved TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否已处理：0-未处理 1-已处理',
    resolved_at TIMESTAMP NULL COMMENT '处理时间',
    PRIMARY KEY (id),
    INDEX idx_resolved (resolved)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='死信队列表';

CREATE TABLE scheduled_jobs (
    id VARCHAR(64) NOT NULL COMMENT '任务 ID',
    name VARCHAR(128) NOT NULL COMMENT '任务名称',
    cron VARCHAR(64) NOT NULL COMMENT 'cron 表达式',
    enabled TINYINT(1) NOT NULL DEFAULT 1 COMMENT '是否启用',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    deleted_at DATETIME DEFAULT NULL COMMENT '软删除时间',
    PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='定时任务定义表';

-- +goose Down
DROP TABLE scheduled_jobs;
DROP TABLE IF EXISTS dead_letters;
DROP TABLE IF EXISTS job_history;
DROP TABLE IF EXISTS jobs;
