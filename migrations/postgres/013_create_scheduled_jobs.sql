-- +goose Up
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

-- +goose Down
DROP TABLE IF EXISTS scheduled_jobs;
