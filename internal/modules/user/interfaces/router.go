package interfaces

import (
	"jimu/internal/modules/user/application"
	"jimu/internal/platform/http/middleware"
	"jimu/internal/shared/pagination"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RegisterUserRoutes 注册用户路由
// rdb 可选：传入 Redis 客户端以启用 POST 幂等性保护
func RegisterUserRoutes(r *gin.RouterGroup, service *application.UserService, rdb ...*redis.Client) {
	handler := NewUserHandler(service)
	users := r.Group("/users")
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
		users.GET("/:id", handler.Get)
		users.PUT("/:id", middleware.ValidateJSON(&application.UpdateUserRequest{}), handler.Update)
		users.DELETE("/:id", handler.Delete)
	}
}
