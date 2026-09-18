package interfaces

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"jimu/internal/platform/ws"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func newWSHandler(t *testing.T) (*ws.ClientHub, *ws.PresenceManager, *AdminWSHandler) {
	t.Helper()
	pm := ws.NewPresenceManager()
	hub := ws.NewClientHub(pm, ws.NewChannelManager())
	return hub, pm, NewAdminWSHandler(hub, pm)
}

func TestAdminWSHandlerPush(t *testing.T) {
	gin.SetMode(gin.TestMode)
	_, _, h := newWSHandler(t)
	r := gin.New()
	r.POST("/push", h.Push)

	// 发送给指定用户
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/push",
		strings.NewReader(`{"user_id":5,"type":"notification","payload":{"title":"hi"}}`)))
	assert.Equal(t, http.StatusOK, w.Code)

	// 发送到频道
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest(http.MethodPost, "/push",
		strings.NewReader(`{"type":"notification","channel":"room:1","payload":{}}`)))
	assert.Equal(t, http.StatusOK, w2.Code)

	// 广播
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, httptest.NewRequest(http.MethodPost, "/push",
		strings.NewReader(`{"type":"notification","payload":{}}`)))
	assert.Equal(t, http.StatusOK, w3.Code)

	// 非法 JSON / 缺 type
	w4 := httptest.NewRecorder()
	r.ServeHTTP(w4, httptest.NewRequest(http.MethodPost, "/push", strings.NewReader(`{bad`)))
	assert.Equal(t, http.StatusBadRequest, w4.Code)

	// hub 未初始化
	pm := ws.NewPresenceManager()
	r2 := gin.New()
	r2.POST("/push", NewAdminWSHandler(nil, pm).Push)
	w5 := httptest.NewRecorder()
	r2.ServeHTTP(w5, httptest.NewRequest(http.MethodPost, "/push",
		strings.NewReader(`{"type":"notification","payload":{}}`)))
	assert.Equal(t, http.StatusInternalServerError, w5.Code)
}

func TestAdminWSHandlerPresence(t *testing.T) {
	gin.SetMode(gin.TestMode)
	_, pm, h := newWSHandler(t)
	r := gin.New()
	r.GET("/presence/:userId", h.Presence)

	// 用户未上线
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/presence/1", nil))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "offline")

	// 用户在线
	pm.Online(7, "conn-1")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest(http.MethodGet, "/presence/7", nil))
	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Contains(t, w2.Body.String(), "online")

	// 非法 id
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, httptest.NewRequest(http.MethodGet, "/presence/abc", nil))
	assert.Equal(t, http.StatusBadRequest, w3.Code)

	// presence 未初始化
	r2 := gin.New()
	r2.GET("/presence/:userId", NewAdminWSHandler(nil, nil).Presence)
	w4 := httptest.NewRecorder()
	r2.ServeHTTP(w4, httptest.NewRequest(http.MethodGet, "/presence/1", nil))
	assert.Equal(t, http.StatusInternalServerError, w4.Code)
}

func TestAdminWSHandlerOnlineUsers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	_, pm, h := newWSHandler(t)
	r := gin.New()
	r.GET("/online", h.OnlineUsers)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/online", nil))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "online_count")

	// 有在线用户
	pm.Online(3, "conn-1")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest(http.MethodGet, "/online", nil))
	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Contains(t, w2.Body.String(), "3")

	// presence 未初始化
	r2 := gin.New()
	r2.GET("/online", NewAdminWSHandler(nil, nil).OnlineUsers)
	w3 := httptest.NewRecorder()
	r2.ServeHTTP(w3, httptest.NewRequest(http.MethodGet, "/online", nil))
	assert.Equal(t, http.StatusInternalServerError, w3.Code)
}
