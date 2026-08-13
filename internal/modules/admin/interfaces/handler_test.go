package interfaces

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"jimu/internal/modules/admin/application"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestHandlerGetErrorCodes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/error-codes", NewHandler(application.NewService("v1", "test", nil)).GetErrorCodes)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/error-codes", nil))
	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]any
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, float64(0), body["code"])
	data, ok := body["data"].([]interface{})
	assert.True(t, ok)
	assert.NotEmpty(t, data)
}
