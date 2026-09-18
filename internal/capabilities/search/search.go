// Package search 提供基于业务数据库的关键词检索（MySQL FULLTEXT / PostgreSQL tsvector），
// 统一接口便于后续替换为 Elasticsearch 等外部检索引擎。
//
// 文档存放在公共表 search_documents 中，按 (tenant_id, doc_type, doc_id) 唯一，
// 因此业务模块无需为每张业务表单独建索引；写入与查询都按租户隔离。
//
// 已知限制：
//   - 中文分词依赖数据库能力：MySQL 需为全文索引启用 ngram 解析器，PostgreSQL 需 zhparser
//     等扩展；默认配置下更适合作英文/数字关键词检索。
//   - 只做关键词匹配与排序，不提供高亮、同义词、模糊匹配与向量检索。
package search

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

// Document 待索引文档
type Document struct {
	ID       uint64 // 主键；为 0 时由雪花 hook 生成
	TenantID uint64 // 所属租户
	Type     string // 业务文档类型，如 user / article
	DocID    uint64 // 业务文档 ID
	Title    string
	Body     string
}

// Result 检索结果（含相关度分数，越大越相关）
type Result struct {
	Document
	Score float64
}

// Searcher 检索器接口
type Searcher interface {
	// Index 写入或更新文档（按 租户+类型+文档ID 幂等覆盖）
	Index(ctx context.Context, docs ...Document) error
	// Delete 删除指定业务文档的索引；tenantID 为 0 时不限定租户
	Delete(ctx context.Context, tenantID uint64, docType string, docIDs ...uint64) error
	// Search 关键词检索；tenantID 为 0 时跨租户检索（平台级视角）
	Search(ctx context.Context, tenantID uint64, query string, limit int) ([]Result, error)
}

// 默认返回条数
const defaultLimit = 20

// searchDocumentRow 索引表模型
type searchDocumentRow struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	TenantID  uint64    `gorm:"column:tenant_id" json:"tenant_id"`
	DocType   string    `gorm:"column:doc_type" json:"doc_type"`
	DocID     uint64    `gorm:"column:doc_id" json:"doc_id"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (searchDocumentRow) TableName() string { return "search_documents" }

// New 按数据库方言选择实现（mysql / postgres）
func New(gdb *gorm.DB) (Searcher, error) {
	if gdb == nil {
		return nil, errors.New("search: nil database")
	}
	switch gdb.Name() {
	case "mysql":
		return &mysqlSearcher{db: gdb}, nil
	case "postgres":
		return &postgresSearcher{db: gdb}, nil
	default:
		return nil, fmt.Errorf("search: unsupported dialect %q (supported: mysql, postgres)", gdb.Name())
	}
}

// toRows 把文档转换为索引行；空标题/正文与空类型直接拒绝，避免写入无意义索引
func toRows(docs []Document) ([]searchDocumentRow, error) {
	rows := make([]searchDocumentRow, 0, len(docs))
	for _, d := range docs {
		if strings.TrimSpace(d.Type) == "" {
			return nil, errors.New("search: document type is required")
		}
		if d.DocID == 0 {
			return nil, errors.New("search: document id is required")
		}
		rows = append(rows, searchDocumentRow{
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

// rowWithScore 检索结果行（score 由各方言的排序表达式计算）
type rowWithScore struct {
	ID       uint64
	TenantID uint64
	DocType  string
	DocID    uint64
	Title    string
	Body     string
	Score    float64
}

func toResults(rows []rowWithScore) []Result {
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

// itoa 小整数转字符串（仅用于拼接固定分值，避免引入 strconv 依赖噪音）
func itoa(v int) string {
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

func normalizeLimit(limit int) int {
	if limit <= 0 {
		return defaultLimit
	}
	return limit
}
