CREATE TABLE IF NOT EXISTS import_jobs (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    type VARCHAR(64) NOT NULL COMMENT 'import type (users/products)',
    filename VARCHAR(255) NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'pending' COMMENT 'pending/processing/completed/failed',
    total_rows INT NOT NULL DEFAULT 0,
    success_rows INT NOT NULL DEFAULT 0,
    error_rows INT NOT NULL DEFAULT 0,
    errors TEXT COMMENT 'JSON error details',
    created_by BIGINT UNSIGNED,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP NULL,
    PRIMARY KEY (id),
    INDEX idx_status (status),
    INDEX idx_created_by (created_by)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
