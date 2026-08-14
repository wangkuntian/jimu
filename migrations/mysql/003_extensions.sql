-- +goose Up
CREATE TABLE IF NOT EXISTS api_keys (
    id BIGINT UNSIGNED NOT NULL COMMENT 'API Key ID',
    name VARCHAR(64) NOT NULL COMMENT 'Key 名称',
    key_prefix VARCHAR(16) NOT NULL COMMENT 'Key 前缀（用于识别）',
    key_hash VARCHAR(64) NOT NULL COMMENT 'SHA-256 哈希',
    scopes TEXT COMMENT '权限范围（JSON 数组）',
    enabled TINYINT(1) NOT NULL DEFAULT 1 COMMENT '是否启用',
    expires_at TIMESTAMP NULL COMMENT '过期时间',
    last_used TIMESTAMP NULL COMMENT '最后使用时间',
    use_count BIGINT NOT NULL DEFAULT 0 COMMENT '使用次数',
    created_by BIGINT UNSIGNED COMMENT '创建者用户 ID',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    PRIMARY KEY (id),
    INDEX idx_key_hash (key_hash),
    INDEX idx_created_by (created_by)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='API 密钥表';

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

CREATE TABLE user_oauth_bindings (
    id BIGINT UNSIGNED NOT NULL COMMENT '主键',
    user_id BIGINT UNSIGNED NOT NULL COMMENT '用户 ID',
    provider VARCHAR(32) NOT NULL COMMENT '提供商（google/github/wechat）',
    subject VARCHAR(128) NOT NULL COMMENT '提供商内唯一 ID',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    deleted_at DATETIME DEFAULT NULL COMMENT '软删除时间',
    PRIMARY KEY (id),
    UNIQUE KEY uk_oauth_binding (provider, subject),
    KEY idx_oauth_user_id (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='第三方登录绑定表';

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
DROP TABLE user_oauth_bindings;
DROP TABLE IF EXISTS import_jobs;
DROP TABLE IF EXISTS dead_letters;
DROP TABLE IF EXISTS job_history;
DROP TABLE IF EXISTS jobs;
DROP TABLE IF EXISTS api_keys;