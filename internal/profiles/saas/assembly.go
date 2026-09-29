// Package saas 是面向外部客户的多租户形态（设计 §2 ②）：minimal + tenant（租户实体/套餐/
// 配额 + 开通式注册）+ audit（审计写入/哈希链）。非 catalog 同 minimal —— 真实邮件渠道由
// 配置 `email.enabled` 打开，不进 import 图。
package saas

import (
	"jimu/internal/assembly"
	accessmodule "jimu/internal/capabilities/access"
	auditmodule "jimu/internal/capabilities/audit"
	authmodule "jimu/internal/capabilities/auth"
	"jimu/internal/capabilities/encryption"
	"jimu/internal/capabilities/notification"
	tenantmodule "jimu/internal/capabilities/tenant"
	"jimu/internal/capabilities/user"
	"jimu/internal/profiles"
)

// Assembly 返回 saas 形态的装配清单。顺序即装配顺序：tenant 先于 access/user/auth
// （后三者经 tenant 端口取配额/开通视图），audit 只读 DB，排最后。
func Assembly() assembly.Assembly {
	return assembly.Assembly{
		Name:    "saas",
		Version: profiles.Version,
		Capabilities: []assembly.Capability{
			{Descriptor: encryption.Descriptor, Wire: encryption.Wire, Ungated: true},
			{Descriptor: notification.Descriptor, Wire: notification.Wire, Ungated: true},
			{Descriptor: tenantmodule.Descriptor, Wire: tenantmodule.Wire},
			{Descriptor: accessmodule.Descriptor, Wire: accessmodule.Wire},
			{Descriptor: user.Descriptor, Wire: user.Wire},
			{Descriptor: authmodule.Descriptor, Wire: authmodule.Wire},
			{Descriptor: auditmodule.Descriptor, Wire: auditmodule.Wire},
		},
		Seed: profiles.StructuralSeed,
	}
}
