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

// AssignRole 分配角色（替换全部角色）
func (h *AdminUserHandler) AssignRole(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, "invalid id"))
		return
	}
	var req struct {
		Roles []string `json:"roles"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, err.Error()))
		return
	}
	if err := h.service.AssignRoles(c.Request.Context(), id, req.Roles); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"assigned": id, "roles": req.Roles})
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
