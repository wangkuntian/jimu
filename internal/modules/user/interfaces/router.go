package interfaces

import (
	"jimu/internal/modules/user/application"

	"github.com/gin-gonic/gin"
)

func RegisterUserRoutes(r *gin.RouterGroup, service *application.UserService) {
	handler := NewUserHandler(service)
	users := r.Group("/users")
	{
		users.POST("", handler.Create)
		users.GET("", handler.List)
		users.GET("/:id", handler.Get)
		users.PUT("/:id", handler.Update)
		users.DELETE("/:id", handler.Delete)
	}
}
