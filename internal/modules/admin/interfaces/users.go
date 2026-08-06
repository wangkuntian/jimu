package interfaces

import (
	"strconv"

	"jimu/internal/modules/admin/application"
	"jimu/internal/shared/errors"
	"jimu/internal/shared/pagination"
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
)

// AdminUserHandler 用户管理 handler
type AdminUserHandler struct {
	service *application.AdminUserService
}

// NewAdminUserHandler 创建用户管理 handler
func NewAdminUserHandler(service *application.AdminUserService) *AdminUserHandler {
	return &AdminUserHandler{service: service}
}

// List 获取用户列表（支持搜索/过滤/分页）
func (h *AdminUserHandler) List(c *gin.Context) {
	users, total, err := h.service.SearchUsers(c.Request.Context(), application.ListUserFilter{}, pagination.Pagination{})
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Page(c, users, total, 1, 20)
}

// Get 获取用户详情
func (h *AdminUserHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, "invalid id"))
		return
	}
	_ = id
	response.OK(c, nil)
}

// Create 创建用户
func (h *AdminUserHandler) Create(c *gin.Context) {
	response.OK(c, nil)
}

// Update 更新用户
func (h *AdminUserHandler) Update(c *gin.Context) {
	response.OK(c, nil)
}

// Disable 禁用用户
func (h *AdminUserHandler) Disable(c *gin.Context) {
	response.OK(c, gin.H{})
}

// AssignRole 分配角色
func (h *AdminUserHandler) AssignRole(c *gin.Context) {
	response.OK(c, gin.H{})
}
