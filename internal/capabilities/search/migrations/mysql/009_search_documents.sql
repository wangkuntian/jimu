-- +goose Up
CREATE TABLE IF NOT EXISTS search_documents (
    id BIGINT UNSIGNED NOT NULL COMMENT '文档ID',
    tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '所属租户ID（0=平台级）',
    doc_type VARCHAR(64) NOT NULL COMMENT '业务文档类型（如 user/article）',
    doc_id BIGINT UNSIGNED NOT NULL COMMENT '业务文档ID',
    title VARCHAR(255) NOT NULL DEFAULT '' COMMENT '标题',
    body MEDIUMTEXT COMMENT '正文（参与全文检索）',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (id),
    UNIQUE INDEX uk_search_doc (tenant_id, doc_type, doc_id),
    INDEX idx_search_updated_at (updated_at),
    FULLTEXT INDEX ft_search_title_body (title, body)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='全文检索文档表';

-- +goose Down
DROP TABLE IF EXISTS search_documents;
