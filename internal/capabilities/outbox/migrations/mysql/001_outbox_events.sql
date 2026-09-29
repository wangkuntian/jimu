-- +goose Up
CREATE TABLE IF NOT EXISTS outbox_events (
    id BIGINT UNSIGNED NOT NULL COMMENT '事件ID',
    aggregate_id VARCHAR(128) NOT NULL COMMENT '聚合根ID',
    event_type VARCHAR(128) NOT NULL COMMENT '事件类型',
    payload JSON NOT NULL COMMENT '事件数据',
    metadata JSON NULL COMMENT '元数据',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    published_at TIMESTAMP NULL DEFAULT NULL COMMENT '发布时间（NULL=未发布）',
    retry_count INT NOT NULL DEFAULT 0 COMMENT '重试次数',
    PRIMARY KEY (id),
    INDEX idx_outbox_aggregate_id (aggregate_id),
    INDEX idx_outbox_event_type (event_type),
    INDEX idx_outbox_published_at (published_at),
    INDEX idx_outbox_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Outbox事件表';

-- +goose Down
DROP TABLE IF EXISTS outbox_events;
