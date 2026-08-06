package interfaces

import (
	"strconv"

	"jimu/internal/modules/permission/application"
	"jimu/internal/shared/errors"
	"jimu/internal/shared/pagination"
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
)

type PermissionHandler struct {
	service *application.PermissionService
}

func NewPermissionHandler(service *application.PermissionService) *PermissionHandler {
	return &PermissionHandler{service: service}
}

// Create godoc
// @Summary      创建权限
// @Description  创建新权限。权限由资源路径（resource）和操作类型（action）唯一标识，用于控制 API 访问。
// @Tags         权限管理
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      application.CreatePermissionRequest  true  "权限信息（名称、资源路径、操作类型）"
// @Success      201   {object}  response.Body  "创建成功，返回权限信息"
// @Failure      400   {object}  contract.ErrorResponse  "参数错误（如名称为空）"
// @Failure      409   {object}  contract.ErrorResponse  "权限已存在（resource + action 重复）"
// @Failure      500   {object}  contract.ErrorResponse  "服务器内部错误"
// @Router       /permissions [post]
func (h *PermissionHandler) Create(c *gin.Context) {
	req := c.MustGet("validated_req").(*application.CreatePermissionRequest)
	perm, err := h.service.Create(c.Request.Context(), *req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, perm)
}

// Get godoc
// @Summary      获取权限详情
// @Description  根据权限 ID 获取权限详细信息，包括资源路径和操作类型。
// @Tags         权限管理
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "权限 ID"
// @Success      200  {object}  response.Body  "成功，返回权限信息"
// @Failure      400  {object}  contract.ErrorResponse  "参数错误（ID 格式无效）"
// @Failure      404  {object}  contract.ErrorResponse  "权限不存在"
// @Failure      500  {object}  contract.ErrorResponse  "服务器内部错误"
// @Router       /permissions/{id} [get]
func (h *PermissionHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, "invalid id"))
		return
	}
	perm, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, perm)
}

// List godoc
// @Summary      获取权限列表
// @Description  分页获取权限列表。支持按 ID、名称、资源路径、操作类型、创建时间排序。
// @Tags         权限管理
// @Produce      json
// @Security     BearerAuth
// @Param        page       query     int     false  "页码（默认 1）"
// @Param        page_size  query     int     false  "每页数量（默认 20，最大 100）"
// @Param        sort       query     string  false  "排序字段（支持 id、name、resource、action、created_at，默认 id）"
// @Param        order      query     string  false  "排序方向（asc 或 desc，默认 desc）"
// @Success      200        {object}  contract.PageResponse  "成功，返回分页权限列表"
// @Failure      400        {object}  contract.ErrorResponse  "参数错误（如无效的排序字段）"
// @Failure      500        {object}  contract.ErrorResponse  "服务器内部错误"
// @Router       /permissions [get]
func (h *PermissionHandler) List(c *gin.Context) {
	p := c.MustGet("validated_query").(*pagination.Pagination)
	if err := p.Normalize("id", "name", "resource", "action", "created_at"); err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, err.Error()))
		return
	}
	perms, total, err := h.service.List(c.Request.Context(), *p)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Page(c, perms, total, p.Page, p.PageSize)
}

// Update godoc
// @Summary      更新权限信息
// @Description  更新指定权限的名称、资源路径或操作类型。
// @Tags         权限管理
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      int                                  true  "权限 ID"
// @Param        body  body      application.UpdatePermissionRequest  true  "要更新的权限信息"
// @Success      200   {object}  response.Body  "更新成功"
// @Failure      400   {object}  contract.ErrorResponse  "参数错误（ID 格式无效）"
// @Failure      404   {object}  contract.ErrorResponse  "权限不存在"
// @Failure      409   {object}  contract.ErrorResponse  "权限冲突（resource + action 重复）"
// @Failure      500   {object}  contract.ErrorResponse  "服务器内部错误"
// @Router       /permissions/{id} [put]
func (h *PermissionHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, "invalid id"))
		return
	}
	req := c.MustGet("validated_req").(*application.UpdatePermissionRequest)
	if err := h.service.Update(c.Request.Context(), id, *req); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, nil)
}

// Delete godoc
// @Summary      删除权限
// @Description  软删除指定权限。删除后权限数据仍保留在数据库中，但无法分配给角色。
// @Tags         权限管理
// @Produce      json
// @Security     BearerAuth
// @Param        id   path  int  true  "权限 ID"
// @Success      204     "删除成功，无返回内容"
// @Failure      400  {object}  contract.ErrorResponse  "参数错误（ID 格式无效）"
// @Failure      500  {object}  contract.ErrorResponse  "服务器内部错误"
// @Router       /permissions/{id} [delete]
func (h *PermissionHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, "invalid id"))
		return
	}
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		response.Fail(c, err)
		return
	}
	response.NoContent(c)
}
