package interfaces

import (
	"strconv"

	"jimu/internal/shared/pagination"

	"github.com/gin-gonic/gin"
)

// paginationFromQuery 从 query 解析分页参数（管理端列表端点共用）。
func paginationFromQuery(c *gin.Context) pagination.Pagination {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	sort := c.DefaultQuery("sort", "id")
	order := c.DefaultQuery("order", "desc")
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return pagination.Pagination{Page: page, PageSize: pageSize, Sort: sort, Order: order}
}
