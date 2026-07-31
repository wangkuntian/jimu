package pagination

import (
	"fmt"
	"strings"
)

const (
	DefaultPage     = 1
	DefaultPageSize = 20
	MaxPageSize     = 100
	DefaultSort     = "id"
	DefaultOrder    = "desc"
)

type Pagination struct {
	Page     int    `form:"page" binding:"omitempty,min=1"`
	PageSize int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	Sort     string `form:"sort"`
	Order    string `form:"order"`
	Filter   string `form:"filter"`
}

func (p *Pagination) Normalize(allowedSorts ...string) error {
	if p.Page == 0 {
		p.Page = DefaultPage
	}
	if p.PageSize == 0 {
		p.PageSize = DefaultPageSize
	}
	if p.PageSize > MaxPageSize {
		p.PageSize = MaxPageSize
	}
	p.Filter = strings.TrimSpace(p.Filter)
	p.Sort = strings.TrimSpace(p.Sort)
	if p.Sort == "" {
		p.Sort = DefaultSort
	}
	if len(allowedSorts) > 0 && !contains(allowedSorts, p.Sort) {
		return fmt.Errorf("invalid sort")
	}
	p.Order = strings.ToLower(strings.TrimSpace(p.Order))
	if p.Order == "" {
		p.Order = DefaultOrder
	}
	if p.Order != "asc" && p.Order != "desc" {
		return fmt.Errorf("invalid order")
	}
	return nil
}

func (p Pagination) GetOffset() int {
	return (p.Page - 1) * p.PageSize
}

func (p Pagination) GetLimit() int {
	return p.PageSize
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
