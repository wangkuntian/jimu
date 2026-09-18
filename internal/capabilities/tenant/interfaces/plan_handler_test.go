package interfaces

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"jimu/internal/capabilities/tenant/application"
	"jimu/internal/capabilities/tenant/domain"
	"jimu/internal/capabilities/tenant/infrastructure"
	"jimu/internal/kernel/tenant"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// newPlanTestRouter 用 sqlite 装配租户 + 套餐路由（含 :id 与静态 usage 路由共存）
func newPlanTestRouter(t *testing.T) (*gin.Engine, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&domain.Tenant{}, &domain.Plan{}))

	// 用量统计涉及的用户/角色/API Key 表由各自模块迁移创建，这里只建最小列结构
	for _, ddl := range []string{
		"CREATE TABLE users (id INTEGER PRIMARY KEY, tenant_id INTEGER, deleted_at DATETIME)",
		"CREATE TABLE roles (id INTEGER PRIMARY KEY, tenant_id INTEGER, deleted_at DATETIME)",
		"CREATE TABLE api_keys (id INTEGER PRIMARY KEY, tenant_id INTEGER)",
	} {
		require.NoError(t, db.Exec(ddl).Error)
	}

	quotaRepo := infrastructure.NewMysqlQuotaRepository(db)
	planService := application.NewPlanService(infrastructure.NewMysqlPlanRepository(db), quotaRepo)
	tenantService := application.NewTenantService(infrastructure.NewMysqlRepository(db))

	r := gin.New()
	// 模拟鉴权中间件注入租户上下文
	r.Use(func(c *gin.Context) {
		ctx := tenant.WithTenant(c.Request.Context(), tenant.DefaultTenantID)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	rg := r.Group("/api/v1")
	RegisterTenantRoutes(rg, tenantService)
	RegisterPlanRoutes(rg, planService)
	return r, db
}

func doJSON(t *testing.T, r *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var reader *strings.Reader
	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func decodeData(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var out struct {
		Code int            `json:"code"`
		Data map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	return out.Data
}

func TestPlanRoutesCreateAssignAndUsage(t *testing.T) {
	r, db := newPlanTestRouter(t)
	require.NoError(t, db.Create(&domain.Tenant{ID: tenant.DefaultTenantID, Code: "default", Name: "默认租户", Status: 1}).Error)
	require.NoError(t, db.Exec("INSERT INTO users (id, tenant_id) VALUES (1, 1)").Error)

	// 创建套餐
	w := doJSON(t, r, http.MethodPost, "/api/v1/tenant-plans",
		`{"code":"FREE","name":"免费版","max_users":2,"max_roles":1,"max_api_keys":0}`)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	data := decodeData(t, w)
	planID, ok := data["id"].(float64)
	require.True(t, ok, "响应缺少套餐 ID: %s", w.Body.String())
	assert.Equal(t, "free", data["code"], "编码统一小写")

	// 分配给默认租户
	w = doJSON(t, r, http.MethodPut, "/api/v1/tenants/1/plan", `{"plan_id":`+itoa(uint64(planID))+`}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// 用量：静态路由 /tenants/usage 必须优先于 /tenants/:id 命中
	w = doJSON(t, r, http.MethodGet, "/api/v1/tenants/usage", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	usage := decodeData(t, w)
	plan, ok := usage["plan"].(map[string]any)
	require.True(t, ok, "usage 应带套餐信息: %s", w.Body.String())
	assert.Equal(t, "free", plan["code"])
	users, ok := usage["users"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, float64(1), users["used"])
	assert.Equal(t, float64(2), users["limit"])
}

func TestPlanRoutesRejectDeleteWhenAssigned(t *testing.T) {
	r, db := newPlanTestRouter(t)
	require.NoError(t, db.Create(&domain.Tenant{ID: tenant.DefaultTenantID, Code: "default", Name: "默认租户", Status: 1}).Error)

	w := doJSON(t, r, http.MethodPost, "/api/v1/tenant-plans", `{"code":"pro","name":"专业版"}`)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	planID := uint64(decodeData(t, w)["id"].(float64))

	w = doJSON(t, r, http.MethodPut, "/api/v1/tenants/1/plan", `{"plan_id":`+itoa(planID)+`}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	w = doJSON(t, r, http.MethodDelete, "/api/v1/tenant-plans/"+itoa(planID), "")
	assert.Equal(t, http.StatusConflict, w.Code, "仍被租户使用的套餐不可删除: %s", w.Body.String())
}

func TestPlanRoutesRequireValidID(t *testing.T) {
	r, _ := newPlanTestRouter(t)
	w := doJSON(t, r, http.MethodDelete, "/api/v1/tenant-plans/not-a-number", "")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// itoa 避免为测试引入 strconv 之外的依赖分支
func itoa(v uint64) string {
	if v == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
}
