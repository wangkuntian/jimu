package http

import (
	_ "jimu/docs/openapi"
	"jimu/internal/platform/http/middleware"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// RegisterSwagger 注册 Swagger UI 路由
func RegisterSwagger(r *gin.RouterGroup) {
	r.GET("/swagger/*any", middleware.Timeout(10*time.Second), ginSwagger.WrapHandler(swaggerFiles.Handler))
}
