package interfaces

import (
	"bytes"
	"encoding/csv"
	"strconv"
	"strings"
	"time"

	"jimu/internal/capabilities/user/application"
	"jimu/internal/capabilities/user/domain"
	"jimu/internal/shared/errors"
	"jimu/internal/shared/pagination"
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	service *application.UserService
}

func NewUserHandler(service *application.UserService) *UserHandler {
	return &UserHandler{service: service}
}

// Create godoc
// @Summary      创建用户
// @Description  创建新用户账户。平台管理员必须指定 tenant_id 并可同时分配 role_ids；租户管理员由后端强制使用当前租户。
// @Tags         用户管理
// @Accept       json
// @Produce      json
// @Param        body  body      application.CreateUserRequest  true  "用户信息、所属租户与角色"
// @Security     BearerAuth
// @Success      201  {object}  application.UserBodyResponse  "创建成功，返回用户信息"
// @Failure      400  {object}  contract.ErrorResponse  "参数错误（如用户名格式不符、密码强度不足）"
// @Failure      409  {object}  contract.ErrorResponse  "用户名已存在"
// @Failure      500  {object}  contract.ErrorResponse  "服务器内部错误"
// @Router       /users [post]
func (h *UserHandler) Create(c *gin.Context) {
	req, _ := c.MustGet("validated_req").(*application.CreateUserRequest)
	user, err := h.service.Create(c.Request.Context(), *req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, user)
}

// Get godoc
// @Summary      获取用户详情
// @Description  根据用户 ID 获取用户详细信息。不包含密码等敏感字段。
// @Tags         用户管理
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "用户 ID"
// @Success      200  {object}  application.UserBodyResponse  "成功，返回用户信息"
// @Failure      400  {object}  contract.ErrorResponse  "参数错误（ID 格式无效）"
// @Failure      404  {object}  contract.ErrorResponse  "用户不存在"
// @Failure      500  {object}  contract.ErrorResponse  "服务器内部错误"
// @Router       /users/{id} [get]
func (h *UserHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, "invalid id"))
		return
	}
	user, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, user)
}

// List godoc
// @Summary      获取用户列表
// @Description  分页获取用户列表。平台管理员可跨租户查询；租户管理员只能看到本租户，范围由后端身份决定。
// @Tags         用户管理
// @Produce      json
// @Security     BearerAuth
// @Param        page       query     int     false  "页码（默认 1）"
// @Param        page_size  query     int     false  "每页数量（默认 20，最大 100）"
// @Param        sort       query     string  false  "排序字段（支持 id、username、created_at，默认 id）"
// @Param        order      query     string  false  "排序方向（asc 或 desc，默认 desc）"
// @Param        username   query     string  false  "用户名子串"
// @Param        status     query     int     false  "状态（0 停用 / 1 启用）"
// @Param        tenant_id  query     string  false  "所属租户 ID（平台级视角可筛选）"
// @Success      200        {object}  application.UserPageResponse  "成功，返回分页用户列表"
// @Failure      400        {object}  contract.ErrorResponse  "参数错误（如无效的排序字段或排序方向）"
// @Failure      500        {object}  contract.ErrorResponse  "服务器内部错误"
// @Router       /users [get]
func (h *UserHandler) List(c *gin.Context) {
	q, _ := c.MustGet("validated_query").(*userListQuery)
	p := q.Pagination
	if err := p.Normalize("id", "username", "created_at"); err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, err.Error()))
		return
	}
	filter := domain.UserListFilter{
		Username: strings.TrimSpace(q.Username),
		Status:   q.Status,
	}
	if q.TenantID != "" {
		tenantID, err := strconv.ParseUint(q.TenantID, 10, 64)
		if err != nil {
			response.Fail(c, errors.New(errors.CodeInvalidParam, "invalid tenant_id"))
			return
		}
		filter.TenantID = tenantID
	}
	users, total, err := h.service.List(c.Request.Context(), p, filter)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Page(c, users, total, p.Page, p.PageSize)
}

// Update godoc
// @Summary      更新用户信息
// @Description  更新指定用户的信息（如状态）。部分字段支持部分更新。
// @Tags         用户管理
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string                         true  "用户 ID"
// @Param        body  body      application.UpdateUserRequest  true  "要更新的用户信息"
// @Success      200   {object}  response.Body  "更新成功"
// @Failure      400   {object}  contract.ErrorResponse  "参数错误（ID 格式无效）"
// @Failure      404   {object}  contract.ErrorResponse  "用户不存在"
// @Failure      500   {object}  contract.ErrorResponse  "服务器内部错误"
// @Router       /users/{id} [put]
func (h *UserHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, "invalid id"))
		return
	}
	req, _ := c.MustGet("validated_req").(*application.UpdateUserRequest)
	if err := h.service.Update(c.Request.Context(), id, *req); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, nil)
}

// Delete godoc
// @Summary      删除用户
// @Description  删除指定用户（软删除，账号从常规查询中消失）。最后一个有效平台超级管理员返回 409；如需保留账号并禁止登录，请使用更新接口将 status 设为 0。
// @Tags         用户管理
// @Produce      json
// @Security     BearerAuth
// @Param        id   path  string  true  "用户 ID"
// @Success      204     "删除成功，无返回内容"
// @Failure      400  {object}  contract.ErrorResponse  "参数错误（ID 格式无效）"
// @Failure      404  {object}  contract.ErrorResponse  "用户不存在"
// @Failure      409  {object}  contract.ErrorResponse  "不能删除最后一个有效平台超级管理员"
// @Failure      500  {object}  contract.ErrorResponse  "服务器内部错误"
// @Router       /users/{id} [delete]
func (h *UserHandler) Delete(c *gin.Context) {
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

// BatchDelete godoc
// @Summary      批量删除用户
// @Description  批量软删除用户（单次最多 100 个）。返回成功/失败计数。
// @Tags         用户管理
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      application.BatchDeleteRequest  true  "要删除的用户 ID 列表"
// @Success      200   {object}  response.Body  "操作完成，返回 success/failed 计数"
// @Failure      400   {object}  contract.ErrorResponse  "参数错误"
// @Router       /users/batch-delete [post]
func (h *UserHandler) BatchDelete(c *gin.Context) {
	req, _ := c.MustGet("validated_req").(*application.BatchDeleteRequest)
	result, err := h.service.BatchDelete(c.Request.Context(), req.IDs)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, result)
}

// ExportCSV godoc
// @Summary      导出用户列表（CSV）
// @Description  导出所有用户为 CSV 文件。列：ID、用户名、状态、创建时间。
// @Tags         用户管理
// @Produce      text/csv
// @Security     BearerAuth
// @Param        sort   query     string  false  "排序字段（支持 id、username、created_at，默认 id）"
// @Param        order  query     string  false  "排序方向（asc 或 desc，默认 desc）"
// @Success      200    {file}    binary  "CSV 文件"
// @Failure      400    {object}  contract.ErrorResponse  "参数错误"
// @Router       /users/export.csv [get]
func (h *UserHandler) ExportCSV(c *gin.Context) {
	p, _ := c.MustGet("validated_query").(*pagination.Pagination)
	if err := p.Normalize("id", "username", "created_at"); err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, err.Error()))
		return
	}
	users, _, err := h.service.List(c.Request.Context(), *p, domain.UserListFilter{})
	if err != nil {
		response.Fail(c, err)
		return
	}
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=users_"+strconv.FormatInt(time.Now().Unix(), 10)+".csv")
	buf := new(bytes.Buffer)
	buf.WriteString("\xEF\xBB\xBF") // UTF-8 BOM
	w := csv.NewWriter(buf)
	_ = w.Write([]string{"ID", "Username", "Status", "Created At"})
	for _, u := range users {
		_ = w.Write([]string{
			u.ID,
			u.Username,
			strconv.FormatInt(int64(u.Status), 10),
			u.CreatedAt.Format(time.RFC3339),
		})
	}
	w.Flush()
	c.String(200, buf.String())
}

type userListQuery struct {
	pagination.Pagination
	Username string `form:"username"`
	Status   *int8  `form:"status" binding:"omitempty,oneof=0 1"`
	TenantID string `form:"tenant_id"`
}
