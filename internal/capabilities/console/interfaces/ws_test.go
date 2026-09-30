package interfaces

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"jimu/internal/contract"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newWSHandler(t *testing.T) (*fakeAdminWebSocket, *AdminWSHandler) {
	t.Helper()
	port := &fakeAdminWebSocket{
		presences:   make(map[uint64]contract.AdminWebSocketPresence),
		connections: make(map[uint64]int),
	}
	return port, NewAdminWSHandler(port)
}

func TestAdminWSHandlerPush(t *testing.T) {
	gin.SetMode(gin.TestMode)
	port, h := newWSHandler(t)
	r := gin.New()
	r.POST("/push", h.Push)

	// 发送给指定用户
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/push",
		strings.NewReader(`{"user_id":5,"type":"notification","payload":{"title":"hi"}}`)))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "user", port.sendMode)
	assert.Equal(t, uint64(5), port.sendUser)
	assert.Equal(t, "notification", port.message.Type)
	assert.JSONEq(t, `{"title":"hi"}`, string(port.message.Payload))
	assert.JSONEq(t, `{"code":0,"message":"ok","data":{"sent":true}}`, w.Body.String())

	// 发送到频道
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest(http.MethodPost, "/push",
		strings.NewReader(`{"type":"notification","channel":"room:1","payload":{}}`)))
	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Equal(t, "channel", port.sendMode)
	assert.Equal(t, "room:1", port.sendChannel)
	assert.Equal(t, "room:1", port.message.Channel)

	// 广播
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, httptest.NewRequest(http.MethodPost, "/push",
		strings.NewReader(`{"type":"notification","payload":{}}`)))
	assert.Equal(t, http.StatusOK, w3.Code)
	assert.Equal(t, "broadcast", port.sendMode)

	// 非法 JSON / 缺 type
	w4 := httptest.NewRecorder()
	r.ServeHTTP(w4, httptest.NewRequest(http.MethodPost, "/push", strings.NewReader(`{bad`)))
	assert.Equal(t, http.StatusBadRequest, w4.Code)

	// hub 未初始化
	r2 := gin.New()
	r2.POST("/push", NewAdminWSHandler(nil).Push)
	w5 := httptest.NewRecorder()
	r2.ServeHTTP(w5, httptest.NewRequest(http.MethodPost, "/push",
		strings.NewReader(`{"type":"notification","payload":{}}`)))
	assert.Equal(t, http.StatusInternalServerError, w5.Code)
}

func TestAdminWSHandlerPresence(t *testing.T) {
	gin.SetMode(gin.TestMode)
	port, h := newWSHandler(t)
	r := gin.New()
	r.GET("/presence/:userId", h.Presence)

	// 用户未上线
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/presence/1", nil))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"code":0,"message":"ok","data":{"user_id":1,"status":"offline","connections":0}}`, w.Body.String())

	// 用户在线
	port.presences[7] = contract.AdminWebSocketPresence{Status: "online"}
	port.connections[7] = 1
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest(http.MethodGet, "/presence/7", nil))
	assert.Equal(t, http.StatusOK, w2.Code)
	assert.JSONEq(t, `{"code":0,"message":"ok","data":{"user_id":7,"status":"online","connections":1}}`, w2.Body.String())

	// 非法 id
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, httptest.NewRequest(http.MethodGet, "/presence/abc", nil))
	assert.Equal(t, http.StatusBadRequest, w3.Code)

	// presence 未初始化
	r2 := gin.New()
	r2.GET("/presence/:userId", NewAdminWSHandler(nil).Presence)
	w4 := httptest.NewRecorder()
	r2.ServeHTTP(w4, httptest.NewRequest(http.MethodGet, "/presence/1", nil))
	assert.Equal(t, http.StatusInternalServerError, w4.Code)
}

func TestAdminWSHandlerOnlineUsers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	port, h := newWSHandler(t)
	r := gin.New()
	r.GET("/online", h.OnlineUsers)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/online", nil))
	assert.Equal(t, http.StatusOK, w.Code)
	var body struct {
		Data struct {
			OnlineCount int      `json:"online_count"`
			Users       []uint64 `json:"users"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Zero(t, body.Data.OnlineCount)

	// 有在线用户
	port.users = []uint64{3}
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest(http.MethodGet, "/online", nil))
	assert.Equal(t, http.StatusOK, w2.Code)
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &body))
	assert.Equal(t, 1, body.Data.OnlineCount)
	assert.Equal(t, []uint64{3}, body.Data.Users)

	// presence 未初始化
	r2 := gin.New()
	r2.GET("/online", NewAdminWSHandler(nil).OnlineUsers)
	w3 := httptest.NewRecorder()
	r2.ServeHTTP(w3, httptest.NewRequest(http.MethodGet, "/online", nil))
	assert.Equal(t, http.StatusInternalServerError, w3.Code)
}
