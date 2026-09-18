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
	// CJK 查询：FULLTEXT 默认分词器按词边界切词，中文/日文/韩文无法命中，
	// 退化为 LIKE 子串匹配（走不了全文索引，注意大表开销；启用 ngram 解析器后可去掉回退）
	if containsCJK(query) {
		return s.searchLike(ctx, tenantID, query, limit)
	}
	// MATCH ... AGAINST 以相关度分数排序（InnoDB 自然语言模式）
	const match = "MATCH(title, body) AGAINST (? IN NATURAL LANGUAGE MODE)"

	var rows []rowWithScore

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

	return toResults(rows), nil
}

// searchLike CJK 回退路径：标题命中权重高于正文，按分值倒序返回
func (s *mysqlSearcher) searchLike(ctx context.Context, tenantID uint64, query string, limit int) ([]Result, error) {
	pattern := likePattern(query)
	var rows []rowWithScore

	q := s.db.WithContext(ctx).Table("search_documents").
		Select("id, tenant_id, doc_type, doc_id, title, body, CASE WHEN title LIKE ? ESCAPE '"+likeEscapeChar+"' THEN "+itoa(cjkFallbackScore)+" ELSE 1 END AS score", pattern).
		Where("title LIKE ? ESCAPE '"+likeEscapeChar+"' OR body LIKE ? ESCAPE '"+likeEscapeChar+"'", pattern, pattern).
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
