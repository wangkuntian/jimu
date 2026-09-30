package interfaces

import (
	"encoding/json"
	"strconv"

	"jimu/internal/contract"
	"jimu/internal/shared/errors"
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
)

// AdminWSHandler WebSocket 管理端点
type AdminWSHandler struct {
	ws contract.AdminWebSocket
}

// NewAdminWSHandler 创建 WebSocket 管理 handler
func NewAdminWSHandler(port contract.AdminWebSocket) *AdminWSHandler {
	return &AdminWSHandler{ws: port}
}

// Push 通过 HTTP 推送 WebSocket 消息（fallback）
func (h *AdminWSHandler) Push(c *gin.Context) {
	var req struct {
		UserID  uint64          `json:"user_id"`
		Type    string          `json:"type" binding:"required"`
		Channel string          `json:"channel"`
		Payload json.RawMessage `json:"payload"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, err.Error()))
		return
	}
	if h.ws == nil {
		response.Fail(c, errors.New(errors.CodeInternalError, "websocket hub not initialized"))
		return
	}
	if req.Payload == nil {
		req.Payload = json.RawMessage("null")
	}
	msg := contract.AdminWebSocketMessage{Type: req.Type, Channel: req.Channel, Payload: req.Payload}
	var sendErr error
	if req.UserID > 0 {
		sendErr = h.ws.SendToUser(req.UserID, msg)
	} else if req.Channel != "" {
		sendErr = h.ws.BroadcastToChannel(req.Channel, msg)
	} else {
		sendErr = h.ws.Broadcast(msg)
	}
	if sendErr != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, sendErr.Error()))
		return
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
	if h.ws == nil {
		response.Fail(c, errors.New(errors.CodeInternalError, "presence manager not initialized"))
		return
	}
	status := "offline"
	if p, ok := h.ws.Presence(userID); ok {
		status = p.Status
	}
	conns := h.ws.Connections(userID)
	response.OK(c, gin.H{"user_id": userID, "status": status, "connections": conns})
}

// OnlineUsers 在线用户列表
func (h *AdminWSHandler) OnlineUsers(c *gin.Context) {
	if h.ws == nil {
		response.Fail(c, errors.New(errors.CodeInternalError, "presence manager not initialized"))
		return
	}
	response.OK(c, gin.H{
		"online_count": h.ws.OnlineCount(),
		"users":        h.ws.OnlineUsers(),
	})
}
