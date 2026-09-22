package oauth

import (
	"jimu/internal/assembly"
	authmodule "jimu/internal/capabilities/auth"
	"jimu/internal/contract"
)

// Wire 装配第三方登录能力：解码 oauth 段、以 auth 段的签发参数构造模块，
// 外部调用复用统一出站 HTTP client。
func Wire(ctx *assembly.Context) (contract.Module, error) {
	cfg := assembly.MustSection[*Config](ctx, ConfigKey)
	if cfg == nil {
		cfg = &Config{}
	}
	return New(ctx.DB(), ctx.Redis(), *cfg, *authConfig(ctx), ctx.HTTPClient()), nil
}

// authConfig 取 auth 段；auth 未启用时该段不加载，回退为零值（旧装配惯例）。
func authConfig(ctx *assembly.Context) *authmodule.Config {
	if cfg := assembly.MustSection[*authmodule.Config](ctx, authmodule.ConfigKey); cfg != nil {
		return cfg
	}
	return &authmodule.Config{}
}
