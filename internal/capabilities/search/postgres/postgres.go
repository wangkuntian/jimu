// Package searchpostgres 提供 PostgreSQL tsvector 检索驱动。
package searchpostgres

import (
	"context"
	"errors"

	"jimu/internal/capabilities/search"
	"jimu/internal/capabilities/search/searchcore"
	"jimu/internal/shared/sqlpattern"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func init() { search.Register("postgres", New) }

// New 构造 PostgreSQL 检索器。
func New(db *gorm.DB) (search.Searcher, error) {
	if db == nil {
		return nil, errors.New("search/postgres: nil database")
	}
	return &searcher{db: db}, nil
}

type searcher struct {
	db *gorm.DB
}

const tsvExpr = "to_tsvector('simple', title || ' ' || coalesce(body, ''))"

func (s *searcher) Index(ctx context.Context, docs ...search.Document) error {
	rows, err := searchcore.ToRows(docs)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "tenant_id"}, {Name: "doc_type"}, {Name: "doc_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"title", "body", "updated_at"}),
	}).Create(&rows).Error
}

func (s *searcher) Delete(ctx context.Context, tenantID uint64, docType string, docIDs ...uint64) error {
	if docType == "" || len(docIDs) == 0 {
		return nil
	}
	q := s.db.WithContext(ctx).Where("doc_type = ? AND doc_id IN ?", docType, docIDs)
	if tenantID != 0 {
		q = q.Where("tenant_id = ?", tenantID)
	}
	return q.Delete(&searchcore.DocumentRow{}).Error
}

func (s *searcher) Search(ctx context.Context, tenantID uint64, query string, limit int) ([]search.Result, error) {
	if query == "" {
		return nil, nil
	}
	if searchcore.ContainsCJK(query) {
		return s.searchLike(ctx, tenantID, query, limit)
	}
	rank := "ts_rank(" + tsvExpr + ", plainto_tsquery('simple', ?))"

	var rows []searchcore.RowWithScore

	q := s.db.WithContext(ctx).Table("search_documents").
		Select("id, tenant_id, doc_type, doc_id, title, body, "+rank+" AS score", query).
		Where(tsvExpr+" @@ plainto_tsquery('simple', ?)", query).
		Order("score DESC").
		Limit(searchcore.NormalizeLimit(limit))
	if tenantID != 0 {
		q = q.Where("tenant_id = ?", tenantID)
	}
	if err := q.Scan(&rows).Error; err != nil {
		return nil, err
	}
	return searchcore.ToResults(rows), nil
}

func (s *searcher) searchLike(ctx context.Context, tenantID uint64, query string, limit int) ([]search.Result, error) {
	pattern := sqlpattern.LikeContains(query)
	var rows []searchcore.RowWithScore

	q := s.db.WithContext(ctx).Table("search_documents").
		Select("id, tenant_id, doc_type, doc_id, title, body, CASE WHEN title ILIKE ? ESCAPE '"+sqlpattern.EscapeChar+"' THEN "+searchcore.Itoa(searchcore.CJKFallbackScore)+" ELSE 1 END AS score", pattern).
		Where("title ILIKE ? ESCAPE '"+sqlpattern.EscapeChar+"' OR body ILIKE ? ESCAPE '"+sqlpattern.EscapeChar+"'", pattern, pattern).
		Order("score DESC, id ASC").
		Limit(searchcore.NormalizeLimit(limit))
	if tenantID != 0 {
		q = q.Where("tenant_id = ?", tenantID)
	}
	if err := q.Scan(&rows).Error; err != nil {
		return nil, err
	}
	return searchcore.ToResults(rows), nil
}
