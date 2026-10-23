package interfaces

import (
	"strconv"

	"jimu/internal/capabilities/access/application"
	"jimu/internal/shared/errors"
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
)

// UserRoleHandler 用户角色分配 handler。
type UserRoleHandler struct {
	service *application.UserRoleService
}

// NewUserRoleHandler 创建用户角色分配 handler。
func NewUserRoleHandler(service *application.UserRoleService) *UserRoleHandler {
	return &UserRoleHandler{service: service}
}

// AssignRole godoc
// @Summary      分配用户角色
// @Description  替换指定用户的全部角色（按角色 ID）。角色必须属于用户所属租户，不存在返回 404。
// @Tags         权限管理
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string  true  "用户 ID"
// @Param        body  body      application.AssignUserRolesRequest  true  "角色 ID 列表"
// @Success      200   {object}  response.Body  "成功"
// @Failure      400   {object}  contract.ErrorResponse  "参数错误"
// @Failure      404   {object}  contract.ErrorResponse  "用户或角色不存在"
// @Router       /users/{id}/roles [post]
func (h *UserRoleHandler) AssignRole(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, "invalid id"))
		return
	}
	var req application.AssignUserRolesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, err.Error()))
		return
	}
	roleIDs, err := parseRoleIDs(req.RoleIDs)
	if err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, "invalid role id"))
		return
	}
	if err := h.service.AssignRoleIDs(c.Request.Context(), id, roleIDs); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"assigned": strconv.FormatUint(id, 10), "role_ids": req.RoleIDs})
}

func parseRoleIDs(values []string) ([]uint64, error) {
	out := make([]uint64, 0, len(values))
	seen := make(map[uint64]struct{}, len(values))
	for _, value := range values {
		id, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out, nil
}
