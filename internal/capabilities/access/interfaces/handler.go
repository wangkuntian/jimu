package interfaces

import (
	"strconv"

	"jimu/internal/capabilities/access/application"
	"jimu/internal/capabilities/access/domain"
	"jimu/internal/kernel/tenant"
	"jimu/internal/shared/errors"
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
// @Description  创建新角色。平台管理员必须传 tenant_id；首版角色接口仅平台管理员可用。租户内角色名唯一。
// @Tags         角色管理
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      application.CreateRoleRequest  true  "角色信息（租户、名称和描述）"
// @Success      201   {object}  application.RoleBodyResponse  "创建成功，返回角色信息"
// @Failure      400   {object}  contract.ErrorResponse  "参数错误（如名称为空）"
// @Failure      403   {object}  contract.ErrorResponse  "仅平台管理员可访问"
// @Failure      404   {object}  contract.ErrorResponse  "租户不存在"
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
// @Description  根据角色 ID 获取角色详细信息。权限列表由 GET /roles/{id}/permissions 单独读取。
// @Tags         角色管理
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "角色 ID"
// @Success      200  {object}  application.RoleBodyResponse  "成功，返回角色信息"
// @Failure      400  {object}  contract.ErrorResponse  "参数错误（ID 格式无效）"
// @Failure      403  {object}  contract.ErrorResponse  "仅平台管理员可访问"
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
// @Param        tenant_id  query     string  false  "目标租户 ID；省略时跨租户查询"
// @Param        keyword    query     string  false  "角色名称模糊匹配"
// @Param        status     query     int     false  "状态（0=停用 1=启用）"
// @Success      200        {object}  application.RolePageResponse  "成功，返回分页角色列表"
// @Failure      400        {object}  contract.ErrorResponse  "参数错误（如无效的排序字段）"
// @Failure      403        {object}  contract.ErrorResponse  "仅平台管理员可访问"
// @Failure      500        {object}  contract.ErrorResponse  "服务器内部错误"
// @Router       /roles [get]
func (h *RoleHandler) List(c *gin.Context) {
	q, _ := c.MustGet("validated_query").(*roleListQuery)
	p := q.Pagination
	if err := p.Normalize("id", "name", "created_at"); err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, err.Error()))
		return
	}
	requestedTenantID := uint64(0)
	if q.TenantID != "" {
		id, err := strconv.ParseUint(q.TenantID, 10, 64)
		if err != nil || id == 0 {
			response.Fail(c, errors.New(errors.CodeInvalidParam, "invalid tenant_id"))
			return
		}
		requestedTenantID = id
	}
	ctxTenantID := tenant.FromContext(c.Request.Context())
	if tenant.IsPlatformScope(c.Request.Context()) {
		ctxTenantID = 0
	}
	if ctxTenantID != 0 && requestedTenantID != 0 && requestedTenantID != ctxTenantID {
		response.Fail(c, errors.New(errors.CodeForbidden, "tenant scope mismatch"))
		return
	}
	if ctxTenantID != 0 {
		requestedTenantID = ctxTenantID
	}
	roles, total, err := h.service.List(c.Request.Context(), p, domain.RoleListFilter{
		TenantID: requestedTenantID,
		Keyword:  q.Keyword,
		Status:   q.Status,
	})
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Page(c, roles, total, p.Page, p.PageSize)
}

// Options godoc
// @Summary      获取可分配角色选项
// @Description  返回指定租户内启用且可分配给用户的角色摘要。租户身份只能读取当前租户。
// @Tags         角色管理
// @Produce      json
// @Security     BearerAuth
// @Param        tenant_id  query     string  false  "目标租户 ID（平台管理员可选）"
// @Success      200        {object}  application.RoleOptionListResponse  "成功，返回角色选项"
// @Failure      400        {object}  contract.ErrorResponse  "参数错误（tenant_id 格式无效）"
// @Failure      403        {object}  contract.ErrorResponse  "跨租户访问"
// @Failure      500        {object}  contract.ErrorResponse  "服务器内部错误"
// @Router       /roles/options [get]
func (h *RoleHandler) Options(c *gin.Context) {
	q, _ := c.MustGet("validated_query").(*roleOptionQuery)
	var tenantID uint64
	if q.TenantID != "" {
		id, err := strconv.ParseUint(q.TenantID, 10, 64)
		if err != nil || id == 0 {
			response.Fail(c, errors.New(errors.CodeInvalidParam, "invalid tenant_id"))
			return
		}
		tenantID = id
	}
	roles, err := h.service.Options(c.Request.Context(), tenantID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, roles)
}

// Update godoc
// @Summary      更新角色信息
// @Description  更新指定角色的名称、描述或状态。tenant_id 不可修改。
// @Tags         角色管理
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string                         true  "角色 ID"
// @Param        body  body      application.UpdateRoleRequest  true  "要更新的角色信息"
// @Success      200   {object}  response.Body  "更新成功"
// @Failure      400   {object}  contract.ErrorResponse  "参数错误（ID 格式无效）"
// @Failure      403   {object}  contract.ErrorResponse  "仅平台管理员可访问"
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
// @Description  永久删除未分配给用户的自定义角色，同时清理其权限关联。系统角色返回 403；角色已分配给用户时返回 409。停用角色请使用更新接口并将 status 设为 0。
// @Tags         角色管理
// @Produce      json
// @Security     BearerAuth
// @Param        id   path  string  true  "角色 ID"
// @Success      204     "删除成功，无返回内容"
// @Failure      400  {object}  contract.ErrorResponse  "参数错误（ID 格式无效）"
// @Failure      403  {object}  contract.ErrorResponse  "仅平台管理员可访问或系统角色不可删除"
// @Failure      404  {object}  contract.ErrorResponse  "角色不存在"
// @Failure      409  {object}  contract.ErrorResponse  "角色已分配给用户"
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
// @Description  整体替换角色权限。权限 ID 使用字符串；空数组表示清空该角色权限。
// @Tags         角色管理
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string                                true  "角色 ID"
// @Param        body  body      application.AssignPermissionsRequest  true  "权限 ID 列表"
// @Success      200   {object}  response.Body  "分配成功"
// @Failure      400   {object}  contract.ErrorResponse  "参数错误（ID 格式无效或权限不存在）"
// @Failure      403   {object}  contract.ErrorResponse  "仅平台管理员可访问或无权授予指定权限"
// @Failure      404   {object}  contract.ErrorResponse  "角色不存在"
// @Failure      500   {object}  contract.ErrorResponse  "服务器内部错误"
// @Router       /roles/{id}/permissions [post]
func (h *RoleHandler) AssignPermissions(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, "invalid id"))
		return
	}
	req, _ := c.MustGet("validated_req").(*application.AssignPermissionsRequest)
	if req.PermissionIDs == nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, "permission_ids is required"))
		return
	}
	permissionIDs, err := parsePermissionIDs(*req.PermissionIDs)
	if err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, "invalid permission id"))
		return
	}
	if err := h.service.AssignPermissions(c.Request.Context(), id, permissionIDs); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, nil)
}

// GetPermissions godoc
// @Summary      获取角色当前权限
// @Description  返回指定角色当前已分配的权限。没有权限时返回空数组。
// @Tags         角色管理
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "角色 ID"
// @Success      200  {object}  application.PermissionListResponse  "成功，返回角色权限"
// @Failure      400  {object}  contract.ErrorResponse  "参数错误（ID 格式无效）"
// @Failure      403  {object}  contract.ErrorResponse  "仅平台管理员可访问"
// @Failure      404  {object}  contract.ErrorResponse  "角色不存在"
// @Failure      500  {object}  contract.ErrorResponse  "服务器内部错误"
// @Router       /roles/{id}/permissions [get]
func (h *RoleHandler) GetPermissions(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, "invalid id"))
		return
	}
	permissions, err := h.service.GetPermissions(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, permissions)
}

func parsePermissionIDs(values []string) ([]uint64, error) {
	out := make([]uint64, 0, len(values))
	for _, value := range values {
		id, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, nil
}
