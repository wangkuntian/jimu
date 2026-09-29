package breach

import (
	"fmt"

	"jimu/internal/assembly"
	authmodule "jimu/internal/capabilities/auth"
	"jimu/internal/contract"
)

// Wire 装配泄露口令检查能力：auth.breach_check_enabled 打开时用统一出站 client 构造
// HIBP k-匿名范围查询检查器并暴露为端口；关闭时端口注册为零值 Checker，auth 取回 nil
// 即降级跳过检查（与旧容器桥接语义一致）。
func Wire(ctx *assembly.Context) (contract.Module, error) {
	authCfg := assembly.MustSection[*authmodule.Config](ctx, authmodule.ConfigKey)
	var checker contract.BreachChecker
	if authCfg != nil && authCfg.BreachCheckEnabled {
		checker = New(ctx.HTTPClient())
	}
	if err := ctx.Provide(PortName, checker); err != nil {
		return nil, fmt.Errorf("provide breach port: %w", err)
	}
	return nil, nil
}
