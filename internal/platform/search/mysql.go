package search

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// mysqlSearcher 基于 MySQL/MariaDB FULLTEXT 的实现
type mysqlSearcher struct {
	db *gorm.DB
}

func (s *mysqlSearcher) Index(ctx context.Context, docs ...Document) error {
	rows, err := toRows(docs)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}
	// ON DUPLICATE KEY UPDATE：按 (tenant_id, doc_type, doc_id) 幂等覆盖
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "tenant_id"}, {Name: "doc_type"}, {Name: "doc_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"title", "body", "updated_at"}),
	}).Create(&rows).Error
}

func (s *mysqlSearcher) Delete(ctx context.Context, tenantID uint64, docType string, docIDs ...uint64) error {
	if docType == "" || len(docIDs) == 0 {
		return nil
	}
	q := s.db.WithContext(ctx).Where("doc_type = ? AND doc_id IN ?", docType, docIDs)
	if tenantID != 0 {
		q = q.Where("tenant_id = ?", tenantID)
	}
	return q.Delete(&searchDocumentRow{}).Error
}

func (s *mysqlSearcher) Search(ctx context.Context, tenantID uint64, query string, limit int) ([]Result, error) {
	if query == "" {
		return nil, nil
	}
	// MATCH ... AGAINST 以相关度分数排序（InnoDB 自然语言模式）
	const match = "MATCH(title, body) AGAINST (? IN NATURAL LANGUAGE MODE)"

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
		Select("id, tenant_id, doc_type, doc_id, title, body, "+match+" AS score", query).
		Where(match, query).
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
