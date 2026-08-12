-- +goose Up
CREATE TABLE IF NOT EXISTS jobs (
    id BIGINT UNSIGNED NOT NULL COMMENT 'Job ID',
    type VARCHAR(64) NOT NULL COMMENT 'Job type (send_email, etc)',
    payload TEXT COMMENT 'JSON payload',
    status VARCHAR(16) NOT NULL DEFAULT 'pending' COMMENT 'pending/running/success/failed/dead',
    priority INT NOT NULL DEFAULT 5 COMMENT '0-9, lower=higher',
    attempts INT NOT NULL DEFAULT 0 COMMENT 'Attempt count',
    max_attempts INT NOT NULL DEFAULT 3 COMMENT 'Max retries',
    next_run_at TIMESTAMP NULL COMMENT 'Next execution time',
    error TEXT COMMENT 'Last error message',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    INDEX idx_status_next_run (status, next_run_at),
    INDEX idx_type (type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Job queue';

CREATE TABLE IF NOT EXISTS job_history (
    id BIGINT UNSIGNED NOT NULL COMMENT 'History ID',
    job_id BIGINT UNSIGNED NOT NULL COMMENT 'Job ID',
    status VARCHAR(16) NOT NULL COMMENT 'success/failed',
    error TEXT COMMENT 'Error message',
    duration_ms BIGINT COMMENT 'Execution duration',
    started_at TIMESTAMP NULL,
    ended_at TIMESTAMP NULL,
    PRIMARY KEY (id),
    INDEX idx_job_id (job_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Job execution history';

CREATE TABLE IF NOT EXISTS dead_letters (
    id BIGINT UNSIGNED NOT NULL COMMENT 'Dead letter ID',
    job_id BIGINT UNSIGNED NOT NULL COMMENT 'Original job ID',
    type VARCHAR(64) NOT NULL COMMENT 'Job type',
    payload TEXT COMMENT 'JSON payload',
    fail_reason TEXT COMMENT 'Failure reason',
    failed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    resolved TINYINT(1) NOT NULL DEFAULT 0,
    resolved_at TIMESTAMP NULL,
    PRIMARY KEY (id),
    INDEX idx_resolved (resolved)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Dead letter queue';

-- +goose Down
DROP TABLE IF EXISTS dead_letters;
DROP TABLE IF EXISTS job_history;
DROP TABLE IF EXISTS jobs;
