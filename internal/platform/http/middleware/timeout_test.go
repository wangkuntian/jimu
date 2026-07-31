package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestTimeoutOnlyPropagatesContextDeadline(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Timeout(5 * time.Millisecond))
	r.GET("/", func(c *gin.Context) {
		if _, ok := c.Request.Context().Deadline(); !ok {
			t.Fatal("request context missing deadline")
		}
		time.Sleep(20 * time.Millisecond)
		c.Status(http.StatusNoContent)
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
}
