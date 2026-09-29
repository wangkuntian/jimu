package interfaces

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"jimu/internal/capabilities/user/application"
	"jimu/internal/capabilities/user/domain"
	apperrors "jimu/internal/shared/errors"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newAdminHandler 构造管理面 handler（复用包内 fakeUserRepository）。
func newAdminHandler() *AdminUserHandler {
	repo := &fakeUserRepository{}
	return NewAdminUserHandler(application.NewAdminUserService(repo, nil))
}

func invokeAdmin(handler gin.HandlerFunc, method, target, body string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Handle(method, "/x", handler)
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestAdminUserList(t *testing.T) {
	w := invokeAdmin(newAdminHandler().List, http.MethodGet, "/x?username=a&status=1", "")
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
}

func TestAdminUserGetRejectsInvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/x", func(c *gin.Context) {
		c.Params = gin.Params{{Key: "id", Value: "abc"}}
		newAdminHandler().Get(c)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, float64(apperrors.CodeInvalidParam), decodeAdminCode(t, w))
}

func TestAdminUserGetOK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/x", func(c *gin.Context) {
		c.Params = gin.Params{{Key: "id", Value: "7"}}
		newAdminHandler().Get(c)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
}

func TestAdminUserCreate(t *testing.T) {
	w := invokeAdmin(newAdminHandler().Create, http.MethodPost, "/x",
		`{"username":"bobby","password":"password123"}`)
	assert.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	// 缺密码 → 参数错误
	w = invokeAdmin(newAdminHandler().Create, http.MethodPost, "/x", `{"username":"bobby"}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAdminUserUpdate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.PUT("/x", func(c *gin.Context) {
		c.Params = gin.Params{{Key: "id", Value: "7"}}
		newAdminHandler().Update(c)
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/x", strings.NewReader(`{"status":0}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// 非法 id（非数字）→ 参数错误
	r2 := gin.New()
	r2.PUT("/x", func(c *gin.Context) {
		c.Params = gin.Params{{Key: "id", Value: "abc"}}
		newAdminHandler().Update(c)
	})
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPut, "/x", strings.NewReader(`{}`))
	req2.Header.Set("Content-Type", "application/json")
	r2.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusBadRequest, w2.Code)
}

func TestAdminUserDisable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.DELETE("/x", func(c *gin.Context) {
		c.Params = gin.Params{{Key: "id", Value: "7"}}
		newAdminHandler().Disable(c)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/x", nil))
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// 非法 id（非数字）→ 参数错误
	r2 := gin.New()
	r2.DELETE("/x", func(c *gin.Context) {
		c.Params = gin.Params{{Key: "id", Value: "abc"}}
		newAdminHandler().Disable(c)
	})
	w2 := httptest.NewRecorder()
	r2.ServeHTTP(w2, httptest.NewRequest(http.MethodDelete, "/x", nil))
	assert.Equal(t, http.StatusBadRequest, w2.Code)
}

// TestRegisterAdminUserRoutes 管理面路由注册（不含 :id/roles，由 access 注册）。
func TestRegisterAdminUserRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterAdminUserRoutes(r.Group("/admin"), application.NewAdminUserService(&fakeUserRepository{}, nil))

	got := map[string]int{}
	for _, route := range r.Routes() {
		got[route.Method+" "+route.Path]++
	}
	for _, want := range []string{
		"GET /admin/users", "POST /admin/users",
		"GET /admin/users/:id", "PUT /admin/users/:id", "DELETE /admin/users/:id",
	} {
		assert.Equal(t, 1, got[want], "路由应恰好注册一次：%s", want)
	}
	// :id/roles 归 access，不得在此重复注册
	assert.Zero(t, got["POST /admin/users/:id/roles"])
}

func decodeAdminCode(t *testing.T, w *httptest.ResponseRecorder) float64 {
	t.Helper()
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	code, _ := body["code"].(float64)
	return code
}

var _ = domain.User{}
