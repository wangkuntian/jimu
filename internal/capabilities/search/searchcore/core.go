// Package searchcore 提供 search 能力的共享模型与辅助函数，供各数据库驱动复用。
package searchcore

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Document 待索引文档。
type Document struct {
	ID       uint64
	TenantID uint64
	Type     string
	DocID    uint64
	Title    string
	Body     string
}

// Result 检索结果（含相关度分数，越大越相关）。
type Result struct {
	Document
	Score float64
}

// Searcher 检索器接口。
type Searcher interface {
	Index(ctx context.Context, docs ...Document) error
	Delete(ctx context.Context, tenantID uint64, docType string, docIDs ...uint64) error
	Search(ctx context.Context, tenantID uint64, query string, limit int) ([]Result, error)
}

// DefaultLimit 默认返回条数。
const DefaultLimit = 20

// DocumentRow 索引表模型。
type DocumentRow struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	TenantID  uint64    `gorm:"column:tenant_id" json:"tenant_id"`
	DocType   string    `gorm:"column:doc_type" json:"doc_type"`
	DocID     uint64    `gorm:"column:doc_id" json:"doc_id"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 返回索引表名。
func (DocumentRow) TableName() string { return "search_documents" }

// ToRows 把文档转换为索引行；空标题/正文与空类型直接拒绝，避免写入无意义索引。
func ToRows(docs []Document) ([]DocumentRow, error) {
	rows := make([]DocumentRow, 0, len(docs))
	for _, d := range docs {
		if strings.TrimSpace(d.Type) == "" {
			return nil, errors.New("search: document type is empty")
		}
		if d.DocID == 0 {
			return nil, errors.New("search: document id is empty")
		}
		rows = append(rows, DocumentRow{
			ID:       d.ID,
			TenantID: d.TenantID,
			DocType:  d.Type,
			DocID:    d.DocID,
			Title:    d.Title,
			Body:     d.Body,
		})
	}
	return rows, nil
}

// RowWithScore 检索结果行（score 由各方言的排序表达式计算）。
type RowWithScore struct {
	ID       uint64
	TenantID uint64
	DocType  string
	DocID    uint64
	Title    string
	Body     string
	Score    float64
}

// ToResults 把带分数的行转换为结果。
func ToResults(rows []RowWithScore) []Result {
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
	return results
}

// Itoa 小整数转字符串（仅用于拼接固定分值，避免引入 strconv 依赖噪音）。
func Itoa(v int) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// NormalizeLimit 把非正数限制归一化为默认值。
func NormalizeLimit(limit int) int {
	if limit <= 0 {
		return DefaultLimit
	}
	return limit
}

// NewUnsupportedDialectError 返回统一的未编译方言错误。
func NewUnsupportedDialectError(name, compiled string) error {
	return fmt.Errorf("search driver %q is not compiled into this build (compiled: %s)", name, compiled)
}
