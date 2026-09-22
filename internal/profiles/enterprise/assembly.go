// Package enterprise 是公司内部系统形态（设计 §2 ③）：单租户（tid=0 平台级视角），
// minimal + console（平台级视图与管理端准入）+ audit + oauth（企业 SSO）+ dataops（导入导出），
// 非 catalog 取 minimal + storage（文件上传落盘）。
package enterprise

import (
	"jimu/internal/assembly"
	accessmodule "jimu/internal/capabilities/access"
	auditmodule "jimu/internal/capabilities/audit"
	authmodule "jimu/internal/capabilities/auth"
	consolemodule "jimu/internal/capabilities/console"
	"jimu/internal/capabilities/dataops"
	"jimu/internal/capabilities/encryption"
	"jimu/internal/capabilities/notification"
	oauthmodule "jimu/internal/capabilities/oauth"
	"jimu/internal/capabilities/storage"
	"jimu/internal/capabilities/user"
	"jimu/internal/profiles"
)

// Assembly 返回 enterprise 形态的装配清单。顺序即装配顺序：storage/notification 先提供
// 端口，access 先于 user，auth 先于 console/oauth（二者经 auth 段签发参数构造），
// audit/dataops 只读 DB，排最后。
func Assembly() assembly.Assembly {
	return assembly.Assembly{
		Name:    "enterprise",
		Version: profiles.Version,
		Capabilities: []assembly.Capability{
			{Descriptor: encryption.Descriptor, Wire: encryption.Wire, Ungated: true},
			{Descriptor: storage.Descriptor, Wire: storage.Wire, Ungated: true,
				Drivers: []string{"local"}},
			{Descriptor: notification.Descriptor, Wire: notification.Wire, Ungated: true},
			{Descriptor: accessmodule.Descriptor, Wire: accessmodule.Wire},
			{Descriptor: user.Descriptor, Wire: user.Wire},
			{Descriptor: authmodule.Descriptor, Wire: authmodule.Wire},
			{Descriptor: auditmodule.Descriptor, Wire: auditmodule.Wire},
			{Descriptor: consolemodule.Descriptor, Wire: consolemodule.Wire},
			{Descriptor: oauthmodule.Descriptor, Wire: oauthmodule.Wire},
			{Descriptor: dataops.Descriptor, Wire: dataops.Wire,
				Drivers: []string{"csv"}},
		},
		Seed: profiles.StructuralSeed,
	}
}
