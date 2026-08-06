package interfaces

import (
	"strconv"

	"jimu/internal/modules/admin/application"
	"jimu/internal/shared/errors"
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
)

// Handler 管理端 handler
type Handler struct {
	service *application.Service
}

// NewHandler 创建管理端 handler
func NewHandler(service *application.Service) *Handler {
	return &Handler{service: service}
}

// GetStatus godoc
// @Summary      获取系统状态
// @Description  获取运行时状态（版本、内存、Goroutine 等）
// @Tags         admin
// @Produce      json
// @Success      200  {object}  response.Body
// @Router       /admin/status [get]
func (h *Handler) GetStatus(c *gin.Context) {
	status := h.service.GetStatus()
	response.OK(c, status)
}

// GetOnlineUsers godoc
// @Summary      获取在线用户列表
// @Description  获取当前在线用户列表
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.Body
// @Router       /admin/users/online [get]
func (h *Handler) GetOnlineUsers(c *gin.Context) {
	users, err := h.service.GetOnlineUsers(c.Request.Context())
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, users)
}

// ForceLogout godoc
// @Summary      强制用户下线
// @Description  强制指定用户的所有会话下线
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Param        user_id  path      int  true  "User ID"
// @Success      200      {object}  response.Body
// @Router       /admin/users/{user_id}/logout [post]
func (h *Handler) ForceLogout(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("user_id"), 10, 64)
	if err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, "invalid user_id"))
		return
	}
	if err := h.service.ForceLogout(c.Request.Context(), userID); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{})
}
