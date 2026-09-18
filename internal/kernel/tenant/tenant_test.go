package tenant

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestNormalizeCode(t *testing.T) {
	if got := NormalizeCode("ACME"); got != "acme" {
		t.Fatalf("NormalizeCode(ACME) = %q, want acme", got)
	}
	if got := NormalizeCode("Sub-B_1"); got != "sub-b_1" {
		t.Fatalf("NormalizeCode(Sub-B_1) = %q, want sub-b_1", got)
	}
}

func TestValidCode(t *testing.T) {
	valid := []string{"a", "acme", "sub-b_1", "t0123456789abcdef", "1", "-_"}
	for _, code := range valid {
		if !ValidCode(code) {
			t.Fatalf("ValidCode(%q) = false, want true", code)
		}
	}
	invalid := []string{"", "has space", "中文", "a/b", strings.Repeat("a", 65)}
	for _, code := range invalid {
		if ValidCode(code) {
			t.Fatalf("ValidCode(%q) = true, want false", code)
		}
	}
}

func TestFromContextWithoutTenant(t *testing.T) {
	if got := FromContext(context.Background()); got != 0 {
		t.Fatalf("FromContext(empty) = %d, want 0", got)
	}
}

func TestWithTenantRoundTrip(t *testing.T) {
	ctx := WithTenant(context.Background(), 42)
	if got := FromContext(ctx); got != 42 {
		t.Fatalf("FromContext = %d, want 42", got)
	}
}

func TestMiddlewareInjectsRequestContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		// 模拟 AuthMiddleware 从 JWT claim 注入 gin context
		c.Set("tenant_id", uint64(7))
		c.Next()
	})
	r.Use(Middleware())

	var got uint64
	r.GET("/probe", func(c *gin.Context) {
		got = FromContext(c.Request.Context())
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/probe", nil))
	if got != 7 {
		t.Fatalf("FromContext = %d, want 7", got)
	}
}

func TestMiddlewareSkipsZeroTenant(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("tenant_id", uint64(0))
		c.Next()
	})
	r.Use(Middleware())

	r.GET("/probe", func(c *gin.Context) {
		if got := FromContext(c.Request.Context()); got != 0 {
			t.Fatalf("FromContext = %d, want 0", got)
		}
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/probe", nil))
}
