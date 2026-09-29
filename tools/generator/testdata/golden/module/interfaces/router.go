package interfaces

import (
	"jimu/internal/capabilities/product/application"
	"jimu/internal/kernel/http/middleware"
	"jimu/internal/shared/pagination"

	"github.com/gin-gonic/gin"
)

func RegisterProductRoutes(r *gin.RouterGroup, service *application.ProductService) {
	handler := NewProductHandler(service)
	group := r.Group("/products")
	{
		group.POST("", middleware.ValidateJSON(&application.CreateProductRequest{}), handler.Create)
		group.GET("", middleware.ValidateQuery(&pagination.Pagination{}), handler.List)
		group.GET("/:id", handler.Get)
		group.PUT("/:id", middleware.ValidateJSON(&application.UpdateProductRequest{}), handler.Update)
		group.DELETE("/:id", handler.Delete)
	}
}
