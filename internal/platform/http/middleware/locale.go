package middleware

import (
	"jimu/internal/shared/i18n"

	"github.com/gin-gonic/gin"
)

// Locale 中间件：从 Accept-Language header 解析语言，注入 gin context
// handler 和 response 层通过 c.GetString("locale") 读取，缺省 zh
func Locale() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("locale", i18n.ParseAcceptLanguage(c.GetHeader("Accept-Language")))
		c.Next()
	}
}
