package oauth

import (
	"fmt"

	"jimu/internal/assembly"
	"jimu/internal/contract"
)

// Wire 装配第三方登录能力：解码 oauth 段、以 auth 段的签发参数构造模块，
// 外部调用复用统一出站 HTTP client。
func Wire(ctx *assembly.Context) (contract.Module, error) {
	cfg := assembly.MustSection[*Config](ctx, ConfigKey)
	if cfg == nil {
		cfg = &Config{}
	}
	users, ok := ctx.Port("user.account").(contract.AccountRepository)
	if !ok {
		return nil, fmt.Errorf("oauth requires user.account port")
	}
	return New(ctx.DB(), ctx.Redis(), *cfg, ctx.AuthConfig(), ctx.HTTPClient(), users), nil
}
