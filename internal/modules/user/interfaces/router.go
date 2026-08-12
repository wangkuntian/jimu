package interfaces

import (
	"time"

	"jimu/internal/modules/user/application"
	"jimu/internal/platform/http/middleware"
	"jimu/internal/shared/pagination"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RegisterUserRoutes 注册用户路由
// rdb 可选：传入 Redis 客户端以启用 POST 幂等性保护 + 用户维度写操作限流
func RegisterUserRoutes(r *gin.RouterGroup, service *application.UserService, rdb ...*redis.Client) {
	handler := NewUserHandler(service)
	users := r.Group("/users")
	{
		// POST 创建用户：可选幂等性保护
		if len(rdb) > 0 && rdb[0] != nil {
			users.POST(
				"",
				middleware.IdempotencyMiddleware(rdb[0], 5),
				middleware.UserRateLimitMiddleware(rdb[0], 60, time.Minute),
				middleware.ValidateJSON(&application.CreateUserRequest{}),
				handler.Create,
			)
		} else {
			users.POST("", middleware.ValidateJSON(&application.CreateUserRequest{}), handler.Create)
		}
		users.GET("", middleware.ValidateQuery(&pagination.Pagination{}), handler.List)
		// CSV 导出（必须在 /:id 之前注册，避免路径冲突）
		users.GET("/export.csv", middleware.ValidateQuery(&pagination.Pagination{}), handler.ExportCSV)
		// 写操作用户维度限流（key 取 AuthMiddleware 注入的 user_id，默认 60 次/分钟）
		if len(rdb) > 0 && rdb[0] != nil {
			userWriteLimit := middleware.UserRateLimitMiddleware(rdb[0], 60, time.Minute)
			users.POST("/batch-delete", middleware.ValidateJSON(&application.BatchDeleteRequest{}), userWriteLimit, handler.BatchDelete)
			users.PUT("/:id", middleware.ValidateJSON(&application.UpdateUserRequest{}), userWriteLimit, handler.Update)
			users.DELETE("/:id", userWriteLimit, handler.Delete)
		} else {
			users.POST("/batch-delete", middleware.ValidateJSON(&application.BatchDeleteRequest{}), handler.BatchDelete)
			users.PUT("/:id", middleware.ValidateJSON(&application.UpdateUserRequest{}), handler.Update)
			users.DELETE("/:id", handler.Delete)
		}
		users.GET("/:id", handler.Get)
	}
}
