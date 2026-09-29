-- +goose Up
CREATE TABLE IF NOT EXISTS search_documents (
    id BIGINT NOT NULL,
    tenant_id BIGINT NOT NULL DEFAULT 0,
    doc_type VARCHAR(64) NOT NULL,
    doc_id BIGINT NOT NULL,
    title VARCHAR(255) NOT NULL DEFAULT '',
    body TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id)
);
CREATE UNIQUE INDEX IF NOT EXISTS uk_search_doc ON search_documents (tenant_id, doc_type, doc_id);
CREATE INDEX IF NOT EXISTS idx_search_updated_at ON search_documents (updated_at);
-- 表达式 GIN 索引：与 capabilities/search 的查询表达式保持一致，无需触发器维护 tsv 列
CREATE INDEX IF NOT EXISTS idx_search_fts ON search_documents
    USING GIN (to_tsvector('simple', title || ' ' || coalesce(body, '')));

COMMENT ON TABLE search_documents IS '全文检索文档表';
COMMENT ON COLUMN search_documents.id IS '文档ID';
COMMENT ON COLUMN search_documents.tenant_id IS '所属租户ID（0=平台级）';
COMMENT ON COLUMN search_documents.doc_type IS '业务文档类型（如 user/article）';
COMMENT ON COLUMN search_documents.doc_id IS '业务文档ID';
COMMENT ON COLUMN search_documents.title IS '标题';
COMMENT ON COLUMN search_documents.body IS '正文（参与全文检索）';
COMMENT ON COLUMN search_documents.created_at IS '创建时间';
COMMENT ON COLUMN search_documents.updated_at IS '更新时间';

-- +goose Down
DROP TABLE IF EXISTS search_documents;
