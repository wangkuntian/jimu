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

func TestPlatformScopeRoundTrip(t *testing.T) {
	ctx := WithPlatformScope(context.Background())
	if !IsPlatformScope(ctx) {
		t.Fatal("IsPlatformScope = false, want true")
	}
	if IsPlatformScope(context.Background()) {
		t.Fatal("IsPlatformScope(empty) = true, want false")
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

func TestMiddlewarePropagatesPlatformScope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("tenant_id", uint64(7))
		c.Set("platform_scope", true)
		c.Next()
	})
	r.Use(Middleware())

	r.GET("/probe", func(c *gin.Context) {
		if got := FromContext(c.Request.Context()); got != 7 {
			t.Fatalf("FromContext = %d, want 7", got)
		}
		if !IsPlatformScope(c.Request.Context()) {
			t.Fatal("IsPlatformScope = false, want true")
		}
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/probe", nil))
}

// T3.1（常量面）默认租户 ID 固定为 1，与迁移 005_tenants.sql 的种子行同值。
// T3.1 的编码面（默认租户 code == "default"）需要读库，落在
// internal/capabilities/tenant/infrastructure/default_tenant_invariant_integration_test.go（Task 8）。
func TestT3_1_DefaultTenantID(t *testing.T) {
	if DefaultTenantID != 1 {
		t.Fatalf("DefaultTenantID = %d, want 1", DefaultTenantID)
	}
}

// T4.1（小写面）NormalizeCode 把大小写混写归一为小写。
func TestT4_1_NormalizeCodeLowercases(t *testing.T) {
	cases := map[string]string{
		"ACME":      "acme",
		"Sub-B_1":   "sub-b_1",
		"Default-1": "default-1",
	}
	for in, want := range cases {
		if got := NormalizeCode(in); got != want {
			t.Errorf("NormalizeCode(%q) = %q, want %q", in, got, want)
		}
	}
}

// T4.1（空白面）实现不做 trim：含空白/换行的编码由 ValidCode 拒绝，不静默接受。
// 口径登记：spec §3 T4.1 写「归一为小写、去空白」，实际契约为「小写归一 + 空白编码被拒」。
func TestT4_1_WhitespacePaddedCodeIsRejectedByValidation(t *testing.T) {
	for _, in := range []string{" acme ", "acme ", " acme", "acme\t", "acme\n"} {
		if code := NormalizeCode(in); ValidCode(code) {
			t.Errorf("ValidCode(NormalizeCode(%q) = %q) = true, want false（空白编码必须被拒绝）", in, code)
		}
	}
}

// T4.2 ValidCode 拒绝空、超长（>64）、非法字符与非 ASCII；边界 64 位合法。
func TestT4_2_ValidCodeRejectsIllegal(t *testing.T) {
	invalid := map[string]string{
		"":                      "空串",
		strings.Repeat("a", 65): "超长（65 位）",
		"a/b":                   "非法字符 /",
		"a.b":                   "非法字符 .",
		"has space":             "内部空白",
		"中文":                    "非 ASCII",
	}
	for code, why := range invalid {
		if ValidCode(code) {
			t.Errorf("ValidCode(%q) = true, want false（%s）", code, why)
		}
	}
	if !ValidCode(strings.Repeat("a", 64)) {
		t.Error("ValidCode(64 位) = false, want true（上限为 64 位）")
	}
}

// forgedCtxKey 非本包的 ctxKey 类型：任何外部键（header/query 派生值会用的形态）
// 都不能被 FromContext 识别 —— 租户只可能来自 WithTenant。
type forgedCtxKey string

// T2.6 断言不存在从 header/query 构造租户的入口：FromContext 只认本包私有的 ctxKey，
// 用外部键类型塞进去的值一律读不到；反向对照证明唯一入口是 WithTenant。
func TestT2_6_ForgedContextKeysDoNotSetTenant(t *testing.T) {
	base := context.Background()
	forged := []context.Context{
		context.WithValue(base, forgedCtxKey("tenant_id"), uint64(9)),
		context.WithValue(base, forgedCtxKey("tid"), uint64(9)),
		context.WithValue(base, forgedCtxKey("X-Tenant-ID"), "9"),
		context.WithValue(base, forgedCtxKey("tenant"), uint64(9)),
	}
	for i, ctx := range forged {
		if got := FromContext(ctx); got != 0 {
			t.Errorf("forged[%d]: FromContext = %d, want 0（只认私有 ctxKey）", i, got)
		}
	}
	if got := FromContext(WithTenant(base, 9)); got != 9 {
		t.Errorf("FromContext(WithTenant(9)) = %d, want 9", got)
	}
}
