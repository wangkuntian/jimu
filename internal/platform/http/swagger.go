package http

import (
	"jimu/internal/platform/http/middleware"
	_ "jimu/docs/openapi"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// RegisterSwagger 注册 Swagger UI 路由
func RegisterSwagger(r *gin.RouterGroup) {
	r.GET("/swagger/*any", middleware.Timeout(10), ginSwagger.WrapHandler(swaggerFiles.Handler))
}
