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
// 注意：传入的 r 已经是 /swagger 组，所以这里用 /*
func RegisterSwagger(r *gin.RouterGroup) {
	r.GET("/*any", middleware.Timeout(10*time.Second), ginSwagger.WrapHandler(swaggerFiles.Handler))
}
