-- +goose Up
CREATE TABLE IF NOT EXISTS api_keys (
    id BIGINT NOT NULL,
    name VARCHAR(64) NOT NULL,
    key_prefix VARCHAR(16) NOT NULL,
    key_hash VARCHAR(64) NOT NULL,
    scopes TEXT,
    enabled SMALLINT NOT NULL DEFAULT 1,
    expires_at TIMESTAMP,
    last_used TIMESTAMP,
    use_count BIGINT NOT NULL DEFAULT 0,
    created_by BIGINT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id)
);
CREATE INDEX IF NOT EXISTS idx_api_keys_key_hash ON api_keys (key_hash);
CREATE INDEX IF NOT EXISTS idx_api_keys_created_by ON api_keys (created_by);

COMMENT ON TABLE api_keys IS 'API 密钥表';
COMMENT ON COLUMN api_keys.id IS 'API Key ID';
COMMENT ON COLUMN api_keys.name IS 'Key 名称';
COMMENT ON COLUMN api_keys.key_prefix IS 'Key 前缀（用于识别）';
COMMENT ON COLUMN api_keys.key_hash IS 'SHA-256 哈希';
COMMENT ON COLUMN api_keys.scopes IS '权限范围（JSON 数组）';
COMMENT ON COLUMN api_keys.enabled IS '是否启用';
COMMENT ON COLUMN api_keys.expires_at IS '过期时间';
COMMENT ON COLUMN api_keys.last_used IS '最后使用时间';
COMMENT ON COLUMN api_keys.use_count IS '使用次数';
COMMENT ON COLUMN api_keys.created_by IS '创建者用户 ID';
COMMENT ON COLUMN api_keys.created_at IS '创建时间';

CREATE TABLE IF NOT EXISTS jobs (
    id BIGINT NOT NULL,
    type VARCHAR(64) NOT NULL,
    payload TEXT,
    status VARCHAR(16) NOT NULL DEFAULT 'pending',
    priority INT NOT NULL DEFAULT 5,
    attempts INT NOT NULL DEFAULT 0,
    max_attempts INT NOT NULL DEFAULT 3,
    next_run_at TIMESTAMP,
    error TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id)
);
CREATE INDEX IF NOT EXISTS idx_jobs_status_next_run ON jobs (status, next_run_at);
CREATE INDEX IF NOT EXISTS idx_jobs_type ON jobs (type);

COMMENT ON TABLE jobs IS 'Job 队列表';
COMMENT ON COLUMN jobs.id IS 'Job ID';
COMMENT ON COLUMN jobs.type IS 'Job 类型（send_email 等）';
COMMENT ON COLUMN jobs.payload IS 'JSON 载荷';
COMMENT ON COLUMN jobs.status IS '状态：pending/running/success/failed/dead';
COMMENT ON COLUMN jobs.priority IS '优先级：0-9，越小越优先';
COMMENT ON COLUMN jobs.attempts IS '已尝试次数';
COMMENT ON COLUMN jobs.max_attempts IS '最大重试次数';
COMMENT ON COLUMN jobs.next_run_at IS '下次执行时间';
COMMENT ON COLUMN jobs.error IS '最近错误信息';
COMMENT ON COLUMN jobs.created_at IS '创建时间';
COMMENT ON COLUMN jobs.updated_at IS '更新时间';

CREATE TABLE IF NOT EXISTS job_history (
    id BIGINT NOT NULL,
    job_id BIGINT NOT NULL,
    status VARCHAR(16) NOT NULL,
    error TEXT,
    duration_ms BIGINT,
    started_at TIMESTAMP,
    ended_at TIMESTAMP,
    PRIMARY KEY (id)
);
CREATE INDEX IF NOT EXISTS idx_job_history_job_id ON job_history (job_id);

COMMENT ON TABLE job_history IS 'Job 执行历史表';
COMMENT ON COLUMN job_history.id IS '历史记录 ID';
COMMENT ON COLUMN job_history.job_id IS 'Job ID';
COMMENT ON COLUMN job_history.status IS '执行结果：success/failed';
COMMENT ON COLUMN job_history.error IS '错误信息';
COMMENT ON COLUMN job_history.duration_ms IS '执行耗时（毫秒）';
COMMENT ON COLUMN job_history.started_at IS '开始时间';
COMMENT ON COLUMN job_history.ended_at IS '结束时间';

CREATE TABLE IF NOT EXISTS dead_letters (
    id BIGINT NOT NULL,
    job_id BIGINT NOT NULL,
    type VARCHAR(64) NOT NULL,
    payload TEXT,
    fail_reason TEXT,
    failed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    resolved SMALLINT NOT NULL DEFAULT 0,
    resolved_at TIMESTAMP,
    PRIMARY KEY (id)
);
CREATE INDEX IF NOT EXISTS idx_dead_letters_resolved ON dead_letters (resolved);

COMMENT ON TABLE dead_letters IS '死信队列表';
COMMENT ON COLUMN dead_letters.id IS '死信 ID';
COMMENT ON COLUMN dead_letters.job_id IS '原 Job ID';
COMMENT ON COLUMN dead_letters.type IS 'Job 类型';
COMMENT ON COLUMN dead_letters.payload IS 'JSON 载荷';
COMMENT ON COLUMN dead_letters.fail_reason IS '失败原因';
COMMENT ON COLUMN dead_letters.failed_at IS '失败时间';
COMMENT ON COLUMN dead_letters.resolved IS '是否已处理：0-未处理 1-已处理';
COMMENT ON COLUMN dead_letters.resolved_at IS '处理时间';

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

CREATE TABLE IF NOT EXISTS user_oauth_bindings (
    id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    provider VARCHAR(32) NOT NULL,
    subject VARCHAR(128) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    PRIMARY KEY (id),
    CONSTRAINT uk_oauth_binding UNIQUE (provider, subject)
);
CREATE INDEX IF NOT EXISTS idx_oauth_user_id ON user_oauth_bindings (user_id);

COMMENT ON TABLE user_oauth_bindings IS '第三方登录绑定表';
COMMENT ON COLUMN user_oauth_bindings.id IS '主键';
COMMENT ON COLUMN user_oauth_bindings.user_id IS '用户 ID';
COMMENT ON COLUMN user_oauth_bindings.provider IS '提供商（google/github/wechat）';
COMMENT ON COLUMN user_oauth_bindings.subject IS '提供商内唯一 ID';
COMMENT ON COLUMN user_oauth_bindings.created_at IS '创建时间';
COMMENT ON COLUMN user_oauth_bindings.updated_at IS '更新时间';
COMMENT ON COLUMN user_oauth_bindings.deleted_at IS '软删除时间';

CREATE TABLE IF NOT EXISTS scheduled_jobs (
    id VARCHAR(64) NOT NULL,
    name VARCHAR(128) NOT NULL,
    cron VARCHAR(64) NOT NULL,
    enabled SMALLINT NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    PRIMARY KEY (id)
);

COMMENT ON TABLE scheduled_jobs IS '定时任务定义表';
COMMENT ON COLUMN scheduled_jobs.id IS '任务 ID';
COMMENT ON COLUMN scheduled_jobs.name IS '任务名称';
COMMENT ON COLUMN scheduled_jobs.cron IS 'cron 表达式';
COMMENT ON COLUMN scheduled_jobs.enabled IS '是否启用';
COMMENT ON COLUMN scheduled_jobs.created_at IS '创建时间';
COMMENT ON COLUMN scheduled_jobs.updated_at IS '更新时间';
COMMENT ON COLUMN scheduled_jobs.deleted_at IS '软删除时间';

-- +goose Down
DROP TABLE IF EXISTS scheduled_jobs;
DROP TABLE IF EXISTS user_oauth_bindings;
DROP TABLE IF EXISTS import_jobs;
DROP TABLE IF EXISTS dead_letters;
DROP TABLE IF EXISTS job_history;
DROP TABLE IF EXISTS jobs;
DROP TABLE IF EXISTS api_keys;
