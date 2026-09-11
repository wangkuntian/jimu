package search

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// postgresSearcher 基于 PostgreSQL tsvector 的实现。
// 使用表达式 GIN 索引（见迁移 009），无需触发器维护 tsv 列。
type postgresSearcher struct {
	db *gorm.DB
}

// tsvExpr 与迁移中的表达式索引保持一致，保证查询能走索引
const tsvExpr = "to_tsvector('simple', title || ' ' || coalesce(body, ''))"

func (s *postgresSearcher) Index(ctx context.Context, docs ...Document) error {
	rows, err := toRows(docs)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}
	// ON CONFLICT：按 (tenant_id, doc_type, doc_id) 幂等覆盖
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "tenant_id"}, {Name: "doc_type"}, {Name: "doc_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"title", "body", "updated_at"}),
	}).Create(&rows).Error
}

func (s *postgresSearcher) Delete(ctx context.Context, tenantID uint64, docType string, docIDs ...uint64) error {
	if docType == "" || len(docIDs) == 0 {
		return nil
	}
	q := s.db.WithContext(ctx).Where("doc_type = ? AND doc_id IN ?", docType, docIDs)
	if tenantID != 0 {
		q = q.Where("tenant_id = ?", tenantID)
	}
	return q.Delete(&searchDocumentRow{}).Error
}

func (s *postgresSearcher) Search(ctx context.Context, tenantID uint64, query string, limit int) ([]Result, error) {
	if query == "" {
		return nil, nil
	}
	rank := "ts_rank(" + tsvExpr + ", plainto_tsquery('simple', ?))"

	var rows []struct {
		ID       uint64
		TenantID uint64
		DocType  string
		DocID    uint64
		Title    string
		Body     string
		Score    float64
	}

	q := s.db.WithContext(ctx).Table("search_documents").
		Select("id, tenant_id, doc_type, doc_id, title, body, "+rank+" AS score", query).
		Where(tsvExpr+" @@ plainto_tsquery('simple', ?)", query).
		Order("score DESC").
		Limit(normalizeLimit(limit))
	if tenantID != 0 {
		q = q.Where("tenant_id = ?", tenantID)
	}
	if err := q.Scan(&rows).Error; err != nil {
		return nil, err
	}

	results := make([]Result, 0, len(rows))
	for _, r := range rows {
		results = append(results, Result{
			Document: Document{
				ID:       r.ID,
				TenantID: r.TenantID,
				Type:     r.DocType,
				DocID:    r.DocID,
				Title:    r.Title,
				Body:     r.Body,
			},
			Score: r.Score,
		})
	}
	return results, nil
}
