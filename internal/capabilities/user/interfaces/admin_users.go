package interfaces

import (
	"strconv"

	"jimu/internal/capabilities/user/application"
	"jimu/internal/shared/errors"
	"jimu/internal/shared/pagination"
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
)

// AdminUserHandler 管理端用户管理 handler。
type AdminUserHandler struct {
	service *application.AdminUserService
}

// NewAdminUserHandler 创建管理端用户 handler。
func NewAdminUserHandler(service *application.AdminUserService) *AdminUserHandler {
	return &AdminUserHandler{service: service}
}

// List 获取用户列表（支持搜索/过滤/分页）
func (h *AdminUserHandler) List(c *gin.Context) {
	username := c.Query("username")
	statusStr := c.Query("status")

	var status *int8
	if statusStr != "" {
		if v, err := strconv.ParseInt(statusStr, 10, 8); err == nil {
			s := int8(v)
			status = &s
		}
	}

	users, total, err := h.service.ListUsers(c.Request.Context(), application.ListUserFilter{
		Username: username,
		Status:   status,
	}, paginationFromQuery(c))
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
	user, err := h.service.GetUser(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, user)
}

// Create 创建用户
func (h *AdminUserHandler) Create(c *gin.Context) {
	var req application.AdminCreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, err.Error()))
		return
	}
	user, err := h.service.CreateUser(c.Request.Context(), req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, user)
}

// Update 更新用户
func (h *AdminUserHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, "invalid id"))
		return
	}
	var req application.AdminUpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, err.Error()))
		return
	}
	if err := h.service.UpdateUser(c.Request.Context(), id, req); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"updated": id})
}

// Disable 禁用用户
func (h *AdminUserHandler) Disable(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, "invalid id"))
		return
	}
	if err := h.service.DisableUser(c.Request.Context(), id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"disabled": id})
}

// paginationFromQuery 从 query 解析分页参数
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

// RegisterAdminUserRoutes 注册管理端用户路由。
// 注意：`POST /users/:id/roles` 由 access 能力注册（user_roles 表所有者），此处不重复。
func RegisterAdminUserRoutes(admin *gin.RouterGroup, service *application.AdminUserService) {
	handler := NewAdminUserHandler(service)
	admin.GET("/users", handler.List)
	admin.POST("/users", handler.Create)
	admin.GET("/users/:id", handler.Get)
	admin.PUT("/users/:id", handler.Update)
	admin.DELETE("/users/:id", handler.Disable)
}
