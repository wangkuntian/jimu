package interfaces

import (
	"strconv"

	"jimu/internal/capabilities/access/application"
	"jimu/internal/shared/errors"
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
)

// UserRoleHandler 用户角色分配 handler（管理端）。
type UserRoleHandler struct {
	service *application.UserRoleService
}

// NewUserRoleHandler 创建用户角色分配 handler。
func NewUserRoleHandler(service *application.UserRoleService) *UserRoleHandler {
	return &UserRoleHandler{service: service}
}

// AssignRole godoc
// @Summary      分配用户角色
// @Description  替换指定用户的全部角色（按角色名）。角色名在用户所属租户内解析，不存在返回 404。
// @Tags         权限管理
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      int     true  "用户 ID"
// @Param        body  body      object  true  "角色名列表"
// @Success      200   {object}  response.Body  "成功"
// @Failure      400   {object}  contract.ErrorResponse  "参数错误"
// @Failure      404   {object}  contract.ErrorResponse  "用户或角色不存在"
// @Router       /admin/users/{id}/roles [post]
func (h *UserRoleHandler) AssignRole(c *gin.Context) {
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
