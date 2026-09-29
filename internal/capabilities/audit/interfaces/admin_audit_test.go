package interfaces

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	auditdomain "jimu/internal/capabilities/audit/domain"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// newSqliteDB 创建内存 sqlite 并迁移给定模型
func newSqliteDB(t *testing.T, models ...interface{}) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(models...))
	return db
}

func TestAdminAuditHandlerList(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newSqliteDB(t, &auditdomain.AuditLog{})
	assert.NoError(t, db.Create(&auditdomain.AuditLog{UserID: 1, Username: "alice", Action: "login", Resource: "user"}).Error)

	r := gin.New()
	r.GET("/audit", NewAdminAuditHandler(db).List)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/audit?page=1&page_size=20", nil))
	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]any
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, float64(0), body["code"])
	assert.Equal(t, float64(1), body["total"])
}
