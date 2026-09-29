package interfaces

import (
	"testing"

	"jimu/internal/capabilities/user/application"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// TestRegisterUserRoutesWithoutRedis 无 Redis 时自助面路由全部注册且恰好一次。
func TestRegisterUserRoutesWithoutRedis(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterUserRoutes(r.Group("/api/v1"), application.NewUserService(&fakeUserRepository{}, nil))

	got := map[string]int{}
	for _, route := range r.Routes() {
		got[route.Method+" "+route.Path]++
	}
	for _, want := range []string{
		"POST /api/v1/users",
		"GET /api/v1/users",
		"GET /api/v1/users/export.csv",
		"POST /api/v1/users/batch-delete",
		"PUT /api/v1/users/:id",
		"DELETE /api/v1/users/:id",
		"GET /api/v1/users/:id",
	} {
		assert.Equal(t, 1, got[want], "路由应恰好注册一次：%s", want)
	}
}
