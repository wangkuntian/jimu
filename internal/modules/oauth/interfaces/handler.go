// internal/modules/oauth/interfaces/handler.go
package interfaces

import (
	"jimu/internal/modules/oauth/application"
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
)

// OAuthHandler OAuth HTTP 处理器
type OAuthHandler struct {
	service *application.OAuthService
}

// NewOAuthHandler 创建 OAuth 处理器
func NewOAuthHandler(service *application.OAuthService) *OAuthHandler {
	return &OAuthHandler{service: service}
}

// Login godoc
// @Summary      OAuth 登录跳转
// @Description  重定向到第三方授权页
// @Tags         OAuth
// @Param        provider path string true "提供商 (google/github/wechat)"
// @Param        state   query string true "防 CSRF 状态"
// @Success      302
// @Router       /oauth/{provider}/login [get]
func (h *OAuthHandler) Login(c *gin.Context) {
	providerName := c.Param("provider")
	url, err := h.service.AuthURL(providerName, c.Query("state"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	c.Redirect(302, url)
}

// Callback godoc
// @Summary      OAuth 回调
// @Description  处理第三方授权回调，签发 JWT
// @Tags         OAuth
// @Param        provider path string true "提供商"
// @Param        code      query string true "授权码"
// @Param        state     query string true "状态"
// @Success      200       {object} response.Body "登录成功"
// @Router       /oauth/{provider}/callback [get]
func (h *OAuthHandler) Callback(c *gin.Context) {
	providerName := c.Param("provider")
	tokenPair, err := h.service.Login(c.Request.Context(), providerName, c.Query("code"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, tokenPair)
}
