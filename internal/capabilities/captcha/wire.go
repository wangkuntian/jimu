package captcha

import (
	"fmt"
	"time"

	"jimu/internal/assembly"
	"jimu/internal/contract"
	"jimu/internal/platform/cachemem"
)

// PortName 验证码能力对外提供的端口名：contract.CaptchaVerifier。
const PortName = "captcha"

// Wire 装配验证码能力：优先用 cache 端口（跨实例有效），没有 cache 时降级到进程内缓存
// 并告警。验证码是一次性短 TTL 状态，单实例降级只影响多副本下的一致性。
func Wire(ctx *assembly.Context) (contract.Module, error) {
	// captcha 启用时其配置段必已按启用集解码（Descriptor.Configs 声明，设计 §8）。
	cfg := assembly.MustSection[*Config](ctx, ConfigKey)
	var store contract.Cache
	if c, ok := ctx.Port(contract.CachePortName).(contract.Cache); ok {
		store = c
	} else {
		ctx.Logger().Warnw("captcha has no cache port; falling back to in-process store")
		store = cachemem.New("jimu")
	}
	mod := New(store, time.Duration(cfg.TTLMin)*time.Minute, cfg.Enabled)
	if err := ctx.Provide(PortName, mod.Service()); err != nil {
		return nil, fmt.Errorf("provide captcha port: %w", err)
	}
	return mod, nil
}
