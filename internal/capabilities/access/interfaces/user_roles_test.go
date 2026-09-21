package interfaces

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"jimu/internal/capabilities/access/application"
	"jimu/internal/kernel/tenant"
	apperrors "jimu/internal/shared/errors"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newUserRoleHandlerDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE users (id INTEGER PRIMARY KEY, tenant_id INTEGER NOT NULL DEFAULT 0)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE roles (id INTEGER PRIMARY KEY, name TEXT NOT NULL, tenant_id INTEGER NOT NULL DEFAULT 0)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE user_roles (user_id INTEGER NOT NULL, role_id INTEGER NOT NULL, PRIMARY KEY (user_id, role_id))`).Error)
	require.NoError(t, db.Exec(`INSERT INTO users (id, tenant_id) VALUES (7, 1)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO roles (id, name, tenant_id) VALUES (1, 'admin', 1)`).Error)
	return db
}

func invokeAssignRole(t *testing.T, db *gorm.DB, idParam, body string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/users/:id/roles", func(c *gin.Context) {
		c.Params = gin.Params{{Key: "id", Value: idParam}}
		c.Request = c.Request.WithContext(tenant.WithTenant(c.Request.Context(), 1))
		NewUserRoleHandler(application.NewUserRoleService(db)).AssignRole(c)
	})
	req := httptest.NewRequest(http.MethodPost, "/users/"+idParam+"/roles", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestAssignRoleHandlerOK(t *testing.T) {
	w := invokeAssignRole(t, newUserRoleHandlerDB(t), "7", `{"roles":["admin"]}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	data := body["data"].(map[string]any)
	assert.Equal(t, float64(7), data["assigned"])
}

func TestAssignRoleHandlerInvalidID(t *testing.T) {
	w := invokeAssignRole(t, newUserRoleHandlerDB(t), "abc", `{"roles":["admin"]}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAssignRoleHandlerInvalidBody(t *testing.T) {
	w := invokeAssignRole(t, newUserRoleHandlerDB(t), "7", `not-json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, float64(apperrors.CodeInvalidParam), bodyCodeOf(t, w))
}

func TestRegisterHTTPAssignRoleRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	admin := r.Group("/api/v1/admin")
	admin.POST("/users/:id/roles", NewUserRoleHandler(application.NewUserRoleService(newUserRoleHandlerDB(t))).AssignRole)
	require.Len(t, r.Routes(), 1)
	assert.Equal(t, "/api/v1/admin/users/:id/roles", r.Routes()[0].Path)
}

func bodyCodeOf(t *testing.T, w *httptest.ResponseRecorder) float64 {
	t.Helper()
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	code, _ := body["code"].(float64)
	return code
}
