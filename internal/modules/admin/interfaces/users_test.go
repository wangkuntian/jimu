package interfaces

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"jimu/internal/modules/admin/application"
	userdomain "jimu/internal/modules/user/domain"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func newUserHandler(fake *fakeUserRepository, db ...*gorm.DB) *AdminUserHandler {
	return NewAdminUserHandler(application.NewAdminUserService(fake, db...))
}

func TestAdminUserHandlerList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/users", newUserHandler(&fakeUserRepository{}).List)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/users?status=1&page=1&page_size=20", nil))
	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, float64(0), body["code"])
	assert.Equal(t, float64(1), body["total"])

	// 仓储错误
	r2 := gin.New()
	r2.GET("/users", newUserHandler(&fakeUserRepository{list: func(ctx context.Context, offset, limit int, sort, order string) ([]userdomain.User, int64, error) {
		return nil, 0, errors.New("db down")
	}}).List)
	w2 := httptest.NewRecorder()
	r2.ServeHTTP(w2, httptest.NewRequest(http.MethodGet, "/users", nil))
	assert.Equal(t, http.StatusInternalServerError, w2.Code)
}

func TestAdminUserHandlerGet(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/users/:id", newUserHandler(&fakeUserRepository{}).Get)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/users/1", nil))
	assert.Equal(t, http.StatusOK, w.Code)

	// 非法 id
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest(http.MethodGet, "/users/abc", nil))
	assert.Equal(t, http.StatusBadRequest, w2.Code)

	// 未找到
	r2 := gin.New()
	r2.GET("/users/:id", newUserHandler(&fakeUserRepository{findByID: func(ctx context.Context, id uint64) (*userdomain.User, error) {
		return nil, gorm.ErrRecordNotFound
	}}).Get)
	w3 := httptest.NewRecorder()
	r2.ServeHTTP(w3, httptest.NewRequest(http.MethodGet, "/users/1", nil))
	assert.Equal(t, http.StatusNotFound, w3.Code)
}

func TestAdminUserHandlerCreate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.POST("/users", newUserHandler(&fakeUserRepository{}).Create)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/users",
		strings.NewReader(`{"username":"bobby","password":"password123"}`)))
	assert.Equal(t, http.StatusCreated, w.Code)

	// 非法 JSON
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{bad`)))
	assert.Equal(t, http.StatusBadRequest, w2.Code)

	// 仓储错误
	r2 := gin.New()
	r2.POST("/users", newUserHandler(&fakeUserRepository{create: func(ctx context.Context, user *userdomain.User) error {
		return errors.New("conflict")
	}}).Create)
	w3 := httptest.NewRecorder()
	r2.ServeHTTP(w3, httptest.NewRequest(http.MethodPost, "/users",
		strings.NewReader(`{"username":"bobby","password":"password123"}`)))
	assert.Equal(t, http.StatusConflict, w3.Code)
}

func TestAdminUserHandlerUpdate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.PUT("/users/:id", newUserHandler(&fakeUserRepository{}).Update)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/users/1",
		strings.NewReader(`{"status":0}`)))
	assert.Equal(t, http.StatusOK, w.Code)

	// 非法 id
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest(http.MethodPut, "/users/abc", strings.NewReader(`{"status":0}`)))
	assert.Equal(t, http.StatusBadRequest, w2.Code)

	// 非法 JSON
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, httptest.NewRequest(http.MethodPut, "/users/1", strings.NewReader(`{bad`)))
	assert.Equal(t, http.StatusBadRequest, w3.Code)

	// 未找到
	r2 := gin.New()
	r2.PUT("/users/:id", newUserHandler(&fakeUserRepository{findByID: func(ctx context.Context, id uint64) (*userdomain.User, error) {
		return nil, gorm.ErrRecordNotFound
	}}).Update)
	w4 := httptest.NewRecorder()
	r2.ServeHTTP(w4, httptest.NewRequest(http.MethodPut, "/users/1", strings.NewReader(`{"status":0}`)))
	assert.Equal(t, http.StatusNotFound, w4.Code)
}

func TestAdminUserHandlerDisable(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.DELETE("/users/:id", newUserHandler(&fakeUserRepository{}).Disable)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/users/1", nil))
	assert.Equal(t, http.StatusOK, w.Code)

	// 非法 id
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest(http.MethodDelete, "/users/abc", nil))
	assert.Equal(t, http.StatusBadRequest, w2.Code)
}

func TestAdminUserHandlerAssignRole(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 成功（db 含 roles 表与 user_roles 表）
	db := newSqliteDB(t, &testUserRole{}, &testRole{})
	assert.NoError(t, db.Create(&testRole{ID: 1, Name: "admin"}).Error)
	r := gin.New()
	r.POST("/users/:id/roles", newUserHandler(&fakeUserRepository{}, db).AssignRole)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/users/1/roles",
		strings.NewReader(`{"roles":["admin"]}`)))
	assert.Equal(t, http.StatusOK, w.Code)

	// 非法 id
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest(http.MethodPost, "/users/abc/roles", strings.NewReader(`{"roles":["admin"]}`)))
	assert.Equal(t, http.StatusBadRequest, w2.Code)

	// 非法 JSON
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, httptest.NewRequest(http.MethodPost, "/users/1/roles", strings.NewReader(`{bad`)))
	assert.Equal(t, http.StatusBadRequest, w3.Code)

	// db 未配置 -> 服务层错误
	r2 := gin.New()
	r2.POST("/users/:id/roles", newUserHandler(&fakeUserRepository{}).AssignRole)
	w4 := httptest.NewRecorder()
	r2.ServeHTTP(w4, httptest.NewRequest(http.MethodPost, "/users/1/roles",
		strings.NewReader(`{"roles":["admin"]}`)))
	assert.Equal(t, http.StatusInternalServerError, w4.Code)
}

func TestPaginationFromQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/u", func(c *gin.Context) {
		p := paginationFromQuery(c)
		assert.Equal(t, 2, p.Page)
		assert.Equal(t, 50, p.PageSize)
		assert.Equal(t, "username", p.Sort)
		assert.Equal(t, "asc", p.Order)
		c.Status(200)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/u?page=2&page_size=50&sort=username&order=asc", nil))
	assert.Equal(t, http.StatusOK, w.Code)

	// 边界：page<1、page_size>100 回退默认
	r2 := gin.New()
	r2.GET("/u", func(c *gin.Context) {
		p := paginationFromQuery(c)
		assert.Equal(t, 1, p.Page)
		assert.Equal(t, 20, p.PageSize)
		c.Status(200)
	})
	w2 := httptest.NewRecorder()
	r2.ServeHTTP(w2, httptest.NewRequest(http.MethodGet, "/u?page=0&page_size=999", nil))
	assert.Equal(t, http.StatusOK, w2.Code)
}
