package interfaces

import (
	"strconv"

	"jimu/internal/platform/ws"
	"jimu/internal/shared/errors"
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
)

// AdminWSHandler WebSocket 管理端点
type AdminWSHandler struct {
	hub *ws.ClientHub
	pm  *ws.PresenceManager
}

// NewAdminWSHandler 创建 WebSocket 管理 handler
func NewAdminWSHandler(hub *ws.ClientHub, pm *ws.PresenceManager) *AdminWSHandler {
	return &AdminWSHandler{hub: hub, pm: pm}
}

// Push 通过 HTTP 推送 WebSocket 消息（fallback）
func (h *AdminWSHandler) Push(c *gin.Context) {
	var req struct {
		UserID  uint64      `json:"user_id"`
		Type    string      `json:"type" binding:"required"`
		Channel string      `json:"channel"`
		Payload interface{} `json:"payload"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, err.Error()))
		return
	}
	if h.hub == nil {
		response.Fail(c, errors.New(errors.CodeInternalError, "websocket hub not initialized"))
		return
	}
	msg, err := ws.NewMessage(req.Type, req.Channel, req.Payload)
	if err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, err.Error()))
		return
	}
	if req.UserID > 0 {
		h.hub.SendToUser(req.UserID, msg)
	} else if req.Channel != "" {
		h.hub.BroadcastToChannel(req.Channel, msg)
	} else {
		h.hub.Broadcast(msg)
	}
	response.OK(c, gin.H{"sent": true})
}

// Presence 查询用户在线状态
func (h *AdminWSHandler) Presence(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("userId"), 10, 64)
	if err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, "invalid user id"))
		return
	}
	if h.pm == nil {
		response.Fail(c, errors.New(errors.CodeInternalError, "presence manager not initialized"))
		return
	}
	status := ws.StatusOffline
	conns := 0
	if p, ok := h.pm.GetPresence(userID); ok {
		status = p.Status
	}
	if h.hub != nil {
		conns = h.hub.GetUserConnections(userID)
	}
	response.OK(c, gin.H{"user_id": userID, "status": status, "connections": conns})
}

// OnlineUsers 在线用户列表
func (h *AdminWSHandler) OnlineUsers(c *gin.Context) {
	if h.pm == nil {
		response.Fail(c, errors.New(errors.CodeInternalError, "presence manager not initialized"))
		return
	}
	response.OK(c, gin.H{
		"online_count": h.pm.OnlineCount(),
		"users":        h.pm.OnlineUsers(),
	})
}
