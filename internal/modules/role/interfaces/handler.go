package interfaces

import (
	"strconv"

	"jimu/internal/modules/role/application"
	"jimu/internal/shared/errors"
	"jimu/internal/shared/pagination"
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
)

type RoleHandler struct {
	service *application.RoleService
}

func NewRoleHandler(service *application.RoleService) *RoleHandler {
	return &RoleHandler{service: service}
}

// Create godoc
// @Summary      创建角色
// @Description  创建新角色。角色名称必须唯一，用于权限管理和访问控制。
// @Tags         角色管理
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      application.CreateRoleRequest  true  "角色信息（名称和描述）"
// @Success      201   {object}  response.Body  "创建成功，返回角色信息"
// @Failure      400   {object}  contract.ErrorResponse  "参数错误（如名称为空）"
// @Failure      409   {object}  contract.ErrorResponse  "角色名称已存在"
// @Failure      500   {object}  contract.ErrorResponse  "服务器内部错误"
// @Router       /roles [post]
func (h *RoleHandler) Create(c *gin.Context) {
	req, _ := c.MustGet("validated_req").(*application.CreateRoleRequest)
	role, err := h.service.Create(c.Request.Context(), *req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, role)
}

// Get godoc
// @Summary      获取角色详情
// @Description  根据角色 ID 获取角色详细信息，包括关联的权限列表。
// @Tags         角色管理
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "角色 ID"
// @Success      200  {object}  response.Body  "成功，返回角色信息"
// @Failure      400  {object}  contract.ErrorResponse  "参数错误（ID 格式无效）"
// @Failure      404  {object}  contract.ErrorResponse  "角色不存在"
// @Failure      500  {object}  contract.ErrorResponse  "服务器内部错误"
// @Router       /roles/{id} [get]
func (h *RoleHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, "invalid id"))
		return
	}
	role, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, role)
}

// List godoc
// @Summary      获取角色列表
// @Description  分页获取角色列表。支持按 ID、名称、创建时间排序。
// @Tags         角色管理
// @Produce      json
// @Security     BearerAuth
// @Param        page       query     int     false  "页码（默认 1）"
// @Param        page_size  query     int     false  "每页数量（默认 20，最大 100）"
// @Param        sort       query     string  false  "排序字段（支持 id、name、created_at，默认 id）"
// @Param        order      query     string  false  "排序方向（asc 或 desc，默认 desc）"
// @Success      200        {object}  contract.PageResponse  "成功，返回分页角色列表"
// @Failure      400        {object}  contract.ErrorResponse  "参数错误（如无效的排序字段）"
// @Failure      500        {object}  contract.ErrorResponse  "服务器内部错误"
// @Router       /roles [get]
func (h *RoleHandler) List(c *gin.Context) {
	p, _ := c.MustGet("validated_query").(*pagination.Pagination)
	if err := p.Normalize("id", "name", "created_at"); err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, err.Error()))
		return
	}
	roles, total, err := h.service.List(c.Request.Context(), *p)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Page(c, roles, total, p.Page, p.PageSize)
}

// Update godoc
// @Summary      更新角色信息
// @Description  更新指定角色的名称或描述。
// @Tags         角色管理
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      int                            true  "角色 ID"
// @Param        body  body      application.UpdateRoleRequest  true  "要更新的角色信息"
// @Success      200   {object}  response.Body  "更新成功"
// @Failure      400   {object}  contract.ErrorResponse  "参数错误（ID 格式无效）"
// @Failure      404   {object}  contract.ErrorResponse  "角色不存在"
// @Failure      409   {object}  contract.ErrorResponse  "角色名称已存在"
// @Failure      500   {object}  contract.ErrorResponse  "服务器内部错误"
// @Router       /roles/{id} [put]
func (h *RoleHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, "invalid id"))
		return
	}
	req, _ := c.MustGet("validated_req").(*application.UpdateRoleRequest)
	if err := h.service.Update(c.Request.Context(), id, *req); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, nil)
}

// Delete godoc
// @Summary      删除角色
// @Description  软删除指定角色。删除后角色数据仍保留在数据库中，但无法分配给用户。
// @Tags         角色管理
// @Produce      json
// @Security     BearerAuth
// @Param        id   path  int  true  "角色 ID"
// @Success      204     "删除成功，无返回内容"
// @Failure      400  {object}  contract.ErrorResponse  "参数错误（ID 格式无效）"
// @Failure      500  {object}  contract.ErrorResponse  "服务器内部错误"
// @Router       /roles/{id} [delete]
func (h *RoleHandler) Delete(c *gin.Context) {
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

// AssignPermissions godoc
// @Summary      分配角色权限
// @Description  为指定角色分配权限。传入权限 ID 数组，角色将被赋予这些权限。原有权限会被替换。
// @Tags         角色管理
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      int                                   true  "角色 ID"
// @Param        body  body      application.AssignPermissionsRequest  true  "权限 ID 列表"
// @Success      200   {object}  response.Body  "分配成功"
// @Failure      400   {object}  contract.ErrorResponse  "参数错误（ID 格式无效或权限不存在）"
// @Failure      500   {object}  contract.ErrorResponse  "服务器内部错误"
// @Router       /roles/{id}/permissions [post]
func (h *RoleHandler) AssignPermissions(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, "invalid id"))
		return
	}
	req, _ := c.MustGet("validated_req").(*application.AssignPermissionsRequest)
	if err := h.service.AssignPermissions(c.Request.Context(), id, req.PermissionIDs); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, nil)
}
