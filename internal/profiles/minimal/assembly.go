// Package minimal 是内部微服务/新项目起点形态（设计 §2 ①）：catalog 取 user/access/auth，
// 非 catalog 取 notification（零配置日志渠道兜底）与 encryption（email/phone 字段级加密）；
// 不含租户、MFA、审计、控制台。
package minimal

import (
	"jimu/internal/assembly"
	accessmodule "jimu/internal/capabilities/access"
	authmodule "jimu/internal/capabilities/auth"
	"jimu/internal/capabilities/encryption"
	"jimu/internal/capabilities/notification"
	"jimu/internal/capabilities/user"
	"jimu/internal/profiles"
)

// Assembly 返回 minimal 形态的装配清单。顺序即装配顺序：encryption/notification 先提供
// Cipher 与 Dispatcher 端口，access 先于 user（user 经 access 端口取角色分配器），auth 最后。
// 缺席的 tenant/mfa/captcha/breach/outbox 端口按软依赖降级（auth 的降级分支）。
func Assembly() assembly.Assembly {
	return assembly.Assembly{
		Name:    "minimal",
		Version: profiles.Version,
		Capabilities: []assembly.Capability{
			{Descriptor: encryption.Descriptor, Wire: encryption.Wire, Ungated: true},
			{Descriptor: notification.Descriptor, Wire: notification.Wire, Ungated: true},
			{Descriptor: accessmodule.Descriptor, Wire: accessmodule.Wire},
			{Descriptor: user.Descriptor, Wire: user.Wire},
			{Descriptor: authmodule.Descriptor, Wire: authmodule.Wire},
		},
		Seed: profiles.StructuralSeed,
	}
}
