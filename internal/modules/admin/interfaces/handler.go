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
// @Description  获取运行时系统状态信息，包括：版本号、运行环境（dev/prod）、启动时间、运行时长、Goroutine 数量、CPU 核心数、内存使用情况（已分配、总分配、系统内存、GC 次数、堆分配、堆使用）。
// @Tags         系统管理
// @Produce      json
// @Success      200  {object}  response.Body  "成功，返回系统状态信息"
// @Router       /admin/status [get]
func (h *Handler) GetStatus(c *gin.Context) {
	status := h.service.GetStatus()
	response.OK(c, status)
}

// GetOnlineUsers godoc
// @Summary      获取在线用户列表
// @Description  获取当前在线用户列表。从 Redis 中扫描所有活跃的会话信息，返回用户 ID、用户名和会话 ID。需要管理员权限。
// @Tags         系统管理
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.Body  "成功，返回在线用户列表"
// @Failure      401  {object}  contract.ErrorResponse  "未认证"
// @Failure      403  {object}  contract.ErrorResponse  "无管理员权限"
// @Failure      500  {object}  contract.ErrorResponse  "服务器内部错误"
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
// @Description  强制指定用户的所有会话下线。删除该用户在 Redis 中的所有会话数据，使其在所有设备上需要重新登录。需要管理员权限。
// @Tags         系统管理
// @Produce      json
// @Security     BearerAuth
// @Param        user_id  path      int  true  "要强制下线的用户 ID"
// @Success      200      {object}  response.Body  "成功"
// @Failure      400      {object}  contract.ErrorResponse  "参数错误（ID 格式无效）"
// @Failure      401      {object}  contract.ErrorResponse  "未认证"
// @Failure      403      {object}  contract.ErrorResponse  "无管理员权限"
// @Failure      500      {object}  contract.ErrorResponse  "服务器内部错误"
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

// GetErrorCodes godoc
// @Summary      获取错误码列表
// @Description  获取所有业务错误码及其对应的 HTTP 状态码映射。用于前端和第三方系统集成时参考。包含错误码、错误消息、HTTP 状态码和分类信息。
// @Tags         系统管理
// @Produce      json
// @Success      200  {object}  response.Body  "成功，返回错误码列表"
// @Router       /admin/error-codes [get]
func (h *Handler) GetErrorCodes(c *gin.Context) {
	response.OK(c, errors.AllErrorCodes())
}
