package middleware

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

type validateStub struct {
	Name string `json:"name" binding:"required"`
}

func TestValidateJSONTranslatesFieldErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name       string
		acceptLang string
		body       string
		wantMsg    string
	}{
		{"zh", "zh-CN,zh;q=0.8", `{}`, "Name 不能为空"},
		{"en", "en-US,en;q=0.9", `{}`, "Name is required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.Use(Locale())
			r.POST("/", ValidateJSON(&validateStub{}), func(c *gin.Context) {
				c.String(200, "ok")
			})
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Accept-Language", tt.acceptLang)
			r.ServeHTTP(w, req)
			if !strings.Contains(w.Body.String(), tt.wantMsg) {
				t.Fatalf("body = %s, want %q", w.Body.String(), tt.wantMsg)
			}
		})
	}
}

type validateJSONFreshStub struct {
	Username       string `json:"username" binding:"required"`
	RememberDevice bool   `json:"remember_device"`
	TenantName     string `json:"tenant_name"`
}

// TestValidateJSONBindsFreshStatePerRequest 钉住 JSON 绑定同样不跨请求串值：
// 与 ValidateQuery 同源 —— dst 在路由注册期只求值一次，复用同一实例时
// 上一次请求出现、本次请求省略的字段会保留旧值（json 解码不会清零缺失字段）。
// 认证侧 loginRequest 带 totp_code / remember_device / tenant_name，
// 上一次的 remember_device=true 或 tenant_name 会泄漏到下一次省略它们的请求。
func TestValidateJSONBindsFreshStatePerRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/", ValidateJSON(&validateJSONFreshStub{}), func(c *gin.Context) {
		req := c.MustGet("validated_req").(*validateJSONFreshStub)
		c.String(http.StatusOK, "%s|%t|%s", req.Username, req.RememberDevice, req.TenantName)
	})

	post := func(body string) string {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		return w.Body.String()
	}

	if got := post(`{"username":"acme","remember_device":true,"tenant_name":"acme"}`); got != "acme|true|acme" {
		t.Fatalf("first request = %q, want acme|true|acme", got)
	}
	if got := post(`{"username":"bob"}`); got != "bob|false|" {
		t.Fatalf("fresh request = %q, want bob|false|（不得继承上一次的 remember_device/tenant_name）", got)
	}
}

type validateQueryStub struct {
	Keyword string `form:"keyword"`
	Status  *int8  `form:"status" binding:"omitempty,oneof=0 1"`
}

// TestValidateQueryBindsFreshStatePerRequest 钉住 query 绑定不跨请求串值：
// 中间件在路由注册期只求值一次 dst，若每次请求复用同一个实例，上一次的
// keyword/status 会泄漏到下一次未携带该参数的请求 —— /tenants 正是靠
// status 指针区分「不传」与 0，串值会让该语义失效。
func TestValidateQueryBindsFreshStatePerRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/", ValidateQuery(&validateQueryStub{}), func(c *gin.Context) {
		q := c.MustGet("validated_query").(*validateQueryStub)
		status := "nil"
		if q.Status != nil {
			status = strconv.Itoa(int(*q.Status))
		}
		c.String(http.StatusOK, "%s|%s", q.Keyword, status)
	})

	get := func(url string) string {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, url, nil))
		return w.Body.String()
	}

	if got := get("/?keyword=acme&status=0"); got != "acme|0" {
		t.Fatalf("first request = %q, want acme|0", got)
	}
	if got := get("/"); got != "|nil" {
		t.Fatalf("fresh request = %q, want |nil（不得继承上一次的 keyword/status）", got)
	}
	if got := get("/?status=1"); got != "|1" {
		t.Fatalf("status-only request = %q, want |1", got)
	}
}
