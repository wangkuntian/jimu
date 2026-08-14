-- +goose Up
CREATE TABLE IF NOT EXISTS outbox_events (
    id BIGINT NOT NULL,
    aggregate_id VARCHAR(128) NOT NULL,
    event_type VARCHAR(128) NOT NULL,
    payload JSON NOT NULL,
    metadata JSON,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    published_at TIMESTAMP,
    retry_count INT NOT NULL DEFAULT 0,
    PRIMARY KEY (id)
);
CREATE INDEX IF NOT EXISTS idx_outbox_aggregate_id ON outbox_events (aggregate_id);
CREATE INDEX IF NOT EXISTS idx_outbox_event_type ON outbox_events (event_type);
CREATE INDEX IF NOT EXISTS idx_outbox_published_at ON outbox_events (published_at);
CREATE INDEX IF NOT EXISTS idx_outbox_created_at ON outbox_events (created_at);

COMMENT ON TABLE outbox_events IS 'Outbox事件表';

-- +goose Down
DROP TABLE IF EXISTS outbox_events;
