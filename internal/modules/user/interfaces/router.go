package interfaces

import (
	"jimu/internal/modules/user/application"
	"jimu/internal/platform/http/middleware"
	"jimu/internal/platform/tenant"
	"jimu/internal/shared/pagination"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RegisterUserRoutes 注册用户路由
// rdb 可选：传入 Redis 客户端以启用 POST 幂等性保护
func RegisterUserRoutes(r *gin.RouterGroup, service *application.UserService, rdb ...*redis.Client) {
	handler := NewUserHandler(service)
	// 多租户隔离：从 X-Tenant-ID header 提取租户上下文
	users := r.Group("/users")
	users.Use(tenant.Middleware("X-Tenant-ID"))
	{
		// POST 创建用户：可选幂等性保护
		if len(rdb) > 0 && rdb[0] != nil {
			users.POST(
				"",
				middleware.IdempotencyMiddleware(rdb[0], 5),
				middleware.ValidateJSON(&application.CreateUserRequest{}),
				handler.Create,
			)
		} else {
			users.POST("", middleware.ValidateJSON(&application.CreateUserRequest{}), handler.Create)
		}
		users.GET("", middleware.ValidateQuery(&pagination.Pagination{}), handler.List)
		// CSV 导出（必须在 /:id 之前注册，避免路径冲突）
		users.GET("/export.csv", middleware.ValidateQuery(&pagination.Pagination{}), handler.ExportCSV)
		users.POST("/batch-delete", middleware.ValidateJSON(&application.BatchDeleteRequest{}), handler.BatchDelete)
		users.GET("/:id", handler.Get)
		users.PUT("/:id", middleware.ValidateJSON(&application.UpdateUserRequest{}), handler.Update)
		users.DELETE("/:id", handler.Delete)
	}
}
