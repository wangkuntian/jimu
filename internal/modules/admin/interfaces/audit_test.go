package interfaces

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	auditdomain "jimu/internal/modules/audit/domain"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

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
