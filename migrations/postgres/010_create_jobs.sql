-- +goose Up
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

COMMENT ON TABLE jobs IS 'Job queue';
COMMENT ON TABLE job_history IS 'Job execution history';
COMMENT ON TABLE dead_letters IS 'Dead letter queue';

-- +goose Down
DROP TABLE IF EXISTS dead_letters;
DROP TABLE IF EXISTS job_history;
DROP TABLE IF EXISTS jobs;
