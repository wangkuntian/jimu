package interfaces

import (
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
)

// AuthConfigResponse 客户端可见的认证能力开关。
// 只暴露渲染注册入口所必需的项：是否需要显示注册入口、
// 以及注册是否等于「开通新租户」（决定 tenant_name 是否必填）。
type AuthConfigResponse struct {
	// PublicRegistration 是否开放公开注册（false 时 /auth/register 路由不注册）
	PublicRegistration bool `json:"public_registration"`
	// ProvisioningEnabled 开通式注册：注册 = 开通新租户，tenant_name 必填
	ProvisioningEnabled bool `json:"provisioning_enabled"`
}

// AuthConfig godoc
// @Summary      获取认证能力开关
// @Description  返回客户端渲染登录/注册入口所需的公开开关。无论是否开放公开注册，本端点都存在。
// @Tags         认证
// @Produce      json
// @Success      200  {object}  AuthConfigResponse  "成功，返回注册相关开关"
// @Router       /auth/config [get]
func (h *AuthHandler) AuthConfig(c *gin.Context) {
	response.OK(c, AuthConfigResponse{
		PublicRegistration:  h.cfg.PublicRegistration,
		ProvisioningEnabled: h.cfg.ProvisioningEnabled,
	})
}
