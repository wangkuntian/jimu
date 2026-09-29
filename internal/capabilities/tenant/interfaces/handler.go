package interfaces

import (
	"strconv"

	"jimu/internal/capabilities/tenant/application"
	"jimu/internal/kernel/http/middleware"
	"jimu/internal/shared/errors"
	"jimu/internal/shared/pagination"
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
)

type TenantHandler struct {
	service *application.TenantService
}

func NewTenantHandler(service *application.TenantService) *TenantHandler {
	return &TenantHandler{service: service}
}

// Create godoc
// @Summary      创建租户
// @Description  创建新租户。编码全局唯一，仅允许字母、数字、短横线和下划线（统一转小写存储）；创建后编码不可修改。
// @Tags         租户管理
// @Accept       json
// @Produce      json
// @Param        body  body      application.CreateTenantRequest  true  "租户信息（编码和名称）"
// @Security     BearerAuth
// @Success      201  {object}  response.Body  "创建成功，返回租户信息"
// @Failure      400  {object}  contract.ErrorResponse  "参数错误（编码格式无效）"
// @Failure      409  {object}  contract.ErrorResponse  "租户编码已存在"
// @Failure      500  {object}  contract.ErrorResponse  "服务器内部错误"
// @Router       /tenants [post]
func (h *TenantHandler) Create(c *gin.Context) {
	req, _ := c.MustGet("validated_req").(*application.CreateTenantRequest)
	t, err := h.service.Create(c.Request.Context(), *req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, t)
}

// List godoc
// @Summary      获取租户列表
// @Description  分页获取租户列表。支持按 ID、编码、创建时间排序。
// @Tags         租户管理
// @Produce      json
// @Security     BearerAuth
// @Param        page       query     int     false  "页码（默认 1）"
// @Param        page_size  query     int     false  "每页数量（默认 20，最大 100）"
// @Param        sort       query     string  false  "排序字段（支持 id、code、created_at，默认 id）"
// @Param        order      query     string  false  "排序方向（asc 或 desc，默认 desc）"
// @Success      200        {object}  contract.PageResponse  "成功，返回分页租户列表"
// @Failure      400        {object}  contract.ErrorResponse  "参数错误（如无效的排序字段）"
// @Failure      500        {object}  contract.ErrorResponse  "服务器内部错误"
// @Router       /tenants [get]
func (h *TenantHandler) List(c *gin.Context) {
	p, _ := c.MustGet("validated_query").(*pagination.Pagination)
	if err := p.Normalize("id", "code", "created_at"); err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, err.Error()))
		return
	}
	tenants, total, err := h.service.List(c.Request.Context(), *p)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Page(c, tenants, total, p.Page, p.PageSize)
}

// Get godoc
// @Summary      获取租户详情
// @Description  根据租户 ID 获取租户详细信息。
// @Tags         租户管理
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "租户 ID"
// @Success      200  {object}  response.Body  "成功，返回租户信息"
// @Failure      400  {object}  contract.ErrorResponse  "参数错误（ID 格式无效）"
// @Failure      404  {object}  contract.ErrorResponse  "租户不存在"
// @Failure      500  {object}  contract.ErrorResponse  "服务器内部错误"
// @Router       /tenants/{id} [get]
func (h *TenantHandler) Get(c *gin.Context) {
	id, err := parseTenantID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	t, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, t)
}

// Update godoc
// @Summary      更新租户信息
// @Description  更新指定租户的名称或状态。编码不可修改。默认租户可改名但不可删除。
// @Tags         租户管理
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      int                            true  "租户 ID"
// @Param        body  body      application.UpdateTenantRequest  true  "要更新的租户信息"
// @Success      200  {object}  response.Body  "更新成功"
// @Failure      400  {object}  contract.ErrorResponse  "参数错误"
// @Failure      404  {object}  contract.ErrorResponse  "租户不存在"
// @Failure      500  {object}  contract.ErrorResponse  "服务器内部错误"
// @Router       /tenants/{id} [put]
func (h *TenantHandler) Update(c *gin.Context) {
	id, err := parseTenantID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	req, _ := c.MustGet("validated_req").(*application.UpdateTenantRequest)
	if err := h.service.Update(c.Request.Context(), id, *req); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, nil)
}

// Delete godoc
// @Summary      删除租户
// @Description  软删除指定租户。默认租户受保护，不可删除。
// @Tags         租户管理
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "租户 ID"
// @Success      200  {object}  response.Body  "删除成功"
// @Failure      400  {object}  contract.ErrorResponse  "参数错误（ID 格式无效）"
// @Failure      404  {object}  contract.ErrorResponse  "租户不存在"
// @Failure      409  {object}  contract.ErrorResponse  "默认租户不可删除"
// @Failure      500  {object}  contract.ErrorResponse  "服务器内部错误"
// @Router       /tenants/{id} [delete]
func (h *TenantHandler) Delete(c *gin.Context) {
	id, err := parseTenantID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, nil)
}

func parseTenantID(c *gin.Context) (uint64, error) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		return 0, errors.New(errors.CodeInvalidParam, "invalid tenant id")
	}
	return id, nil
}

// RegisterTenantRoutes 注册租户路由
func RegisterTenantRoutes(r *gin.RouterGroup, service *application.TenantService) {
	handler := NewTenantHandler(service)
	tenants := r.Group("/tenants")
	{
		tenants.POST("", middleware.ValidateJSON(&application.CreateTenantRequest{}), handler.Create)
		tenants.GET("", middleware.ValidateQuery(&pagination.Pagination{}), handler.List)
		tenants.PUT("/:id", middleware.ValidateJSON(&application.UpdateTenantRequest{}), handler.Update)
		tenants.DELETE("/:id", handler.Delete)
		tenants.GET("/:id", handler.Get)
	}
}
