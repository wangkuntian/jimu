package interfaces

import (
	"jimu/internal/modules/user/application"
	"jimu/internal/platform/http/middleware"
	"jimu/internal/shared/pagination"

	"github.com/gin-gonic/gin"
)

func RegisterUserRoutes(r *gin.RouterGroup, service *application.UserService) {
	handler := NewUserHandler(service)
	users := r.Group("/users")
	{
		users.POST("", middleware.ValidateJSON(&application.CreateUserRequest{}), handler.Create)
		users.GET("", middleware.ValidateQuery(&pagination.Pagination{}), handler.List)
		users.GET("/:id", handler.Get)
		users.PUT("/:id", middleware.ValidateJSON(&application.UpdateUserRequest{}), handler.Update)
		users.DELETE("/:id", handler.Delete)
	}
}
