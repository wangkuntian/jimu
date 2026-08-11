package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	appErrs "jimu/internal/shared/errors"

	"github.com/gin-gonic/gin"
)

func TestOKIncludesStableEnvelopeAndRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/", func(c *gin.Context) {
		c.Set("request_id", "rid-1")
		OK(c, gin.H{"name": "alice"})
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["code"].(float64) != 0 || body["message"] != "ok" || body["request_id"] != "rid-1" {
		t.Fatalf("body = %#v", body)
	}
	if _, ok := body["data"].(map[string]any); !ok {
		t.Fatalf("data missing: %#v", body)
	}
}

func TestPageIncludesStableEnvelopeAndPagination(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/", func(c *gin.Context) {
		c.Set("request_id", "rid-2")
		Page(c, []string{"a"}, 10, 2, 5)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["total"].(float64) != 10 || body["page"].(float64) != 2 || body["page_size"].(float64) != 5 || body["request_id"] != "rid-2" {
		t.Fatalf("body = %#v", body)
	}
}

func TestFailDoesNotLeakInfrastructureDetails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/", func(c *gin.Context) {
		c.Set("request_id", "rid-3")
		Fail(c, appErrs.Wrap(appErrs.CodeInternalError, "sql: password=secret", assertErr("dsn /tmp/secret")))
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", w.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["code"].(float64) != appErrs.CodeInternalError || body["message"] != "服务器内部错误" || body["request_id"] != "rid-3" {
		t.Fatalf("body = %#v", body)
	}
	if _, ok := body["cause"]; ok {
		t.Fatalf("cause leaked: %#v", body)
	}
}

type assertErr string

func (e assertErr) Error() string { return string(e) }
