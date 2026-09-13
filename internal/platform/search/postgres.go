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
	// CJK 查询：tsvector 的 simple 分词器按词边界切词，中文/日文/韩文无法命中，
	// 退化为 ILIKE 子串匹配（走不了 GIN 索引，注意大表开销；安装 zhparser 后可去掉回退）
	if containsCJK(query) {
		return s.searchLike(ctx, tenantID, query, limit)
	}

	rank := "ts_rank(" + tsvExpr + ", plainto_tsquery('simple', ?))"

	var rows []rowWithScore

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

	return toResults(rows), nil
}

// searchLike CJK 回退路径：标题命中权重高于正文，按分值倒序返回
func (s *postgresSearcher) searchLike(ctx context.Context, tenantID uint64, query string, limit int) ([]Result, error) {
	pattern := likePattern(query)
	var rows []rowWithScore

	q := s.db.WithContext(ctx).Table("search_documents").
		Select("id, tenant_id, doc_type, doc_id, title, body, CASE WHEN title ILIKE ? ESCAPE '"+likeEscapeChar+"' THEN "+itoa(cjkFallbackScore)+" ELSE 1 END AS score", pattern).
		Where("title ILIKE ? ESCAPE '"+likeEscapeChar+"' OR body ILIKE ? ESCAPE '"+likeEscapeChar+"'", pattern, pattern).
		Order("score DESC, id ASC").
		Limit(normalizeLimit(limit))
	if tenantID != 0 {
		q = q.Where("tenant_id = ?", tenantID)
	}
	if err := q.Scan(&rows).Error; err != nil {
		return nil, err
	}
	return toResults(rows), nil
}
