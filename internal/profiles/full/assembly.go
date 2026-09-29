// Package full 是完整形态（full profile）的能力清单：catalog 全量 18 项 + 7 个非 catalog
// 条目，与迁移/权限聚合的全量清单逐值一致，是全功能基准与零退化护栏。
package full

import (
	"jimu/internal/assembly"
	accessmodule "jimu/internal/capabilities/access"
	"jimu/internal/capabilities/apidocs"
	"jimu/internal/capabilities/apikey"
	auditmodule "jimu/internal/capabilities/audit"
	authmodule "jimu/internal/capabilities/auth"
	"jimu/internal/capabilities/breach"
	"jimu/internal/capabilities/captcha"
	consolemodule "jimu/internal/capabilities/console"
	"jimu/internal/capabilities/dataops"
	"jimu/internal/capabilities/encryption"
	"jimu/internal/capabilities/feature"
	grpcpkg "jimu/internal/capabilities/grpc"
	mfamodule "jimu/internal/capabilities/mfa"
	"jimu/internal/capabilities/notification"
	oauthmodule "jimu/internal/capabilities/oauth"
	"jimu/internal/capabilities/outbox"
	passkeymodule "jimu/internal/capabilities/passkey"
	"jimu/internal/capabilities/queue"
	"jimu/internal/capabilities/retention"
	"jimu/internal/capabilities/search"
	"jimu/internal/capabilities/storage"
	tenantmodule "jimu/internal/capabilities/tenant"
	"jimu/internal/capabilities/uploadsec"
	"jimu/internal/capabilities/user"
	"jimu/internal/capabilities/ws"
	"jimu/internal/profiles"
)

// Assembly 返回 full 形态的装配清单。能力件一律由 capabilities/<name>/wire.go 自装配，
// 组合根只声明清单，不构造任何能力件。
//
// 顺序即装配顺序：提供端口的能力必须排在消费它的能力之前（encryption/storage/
// notification/queue/outbox/breach 先于 tenant/user/auth/uploadsec/grpc；
// tenant/access 先于 user；captcha/mfa 先于 auth）。
//
// 非 catalog 条目（encryption/storage/notification/retention/apidocs/grpc/ws）标记
// Ungated：由本清单决定是否装配，不受 capabilities.enabled 门控（P2.4 裁定 7）。
func Assembly() assembly.Assembly {
	return assembly.Assembly{
		Name:    "full",
		Version: profiles.Version,
		Capabilities: []assembly.Capability{
			{Descriptor: encryption.Descriptor, Wire: encryption.Wire, Ungated: true},
			{Descriptor: storage.Descriptor, Wire: storage.Wire, Ungated: true,
				Drivers: []string{"local", "s3"}},
			{Descriptor: notification.Descriptor, Wire: notification.Wire, Ungated: true},
			{Descriptor: queue.Descriptor, Wire: queue.Wire,
				Drivers: []string{"redis", "kafka", "rabbitmq"}},
			{Descriptor: outbox.Descriptor, Wire: outbox.Wire},
			{Descriptor: breach.Descriptor, Wire: breach.Wire},
			{Descriptor: tenantmodule.Descriptor, Wire: tenantmodule.Wire},
			{Descriptor: accessmodule.Descriptor, Wire: accessmodule.Wire},
			{Descriptor: user.Descriptor, Wire: user.Wire},
			{Descriptor: captcha.Descriptor, Wire: captcha.Wire},
			{Descriptor: mfamodule.Descriptor, Wire: mfamodule.Wire},
			{Descriptor: authmodule.Descriptor, Wire: authmodule.Wire},
			{Descriptor: passkeymodule.Descriptor, Wire: passkeymodule.Wire},
			{Descriptor: auditmodule.Descriptor, Wire: auditmodule.Wire},
			{Descriptor: consolemodule.Descriptor, Wire: consolemodule.Wire},
			{Descriptor: oauthmodule.Descriptor, Wire: oauthmodule.Wire},
			{Descriptor: apikey.Descriptor, Wire: apikey.Wire},
			{Descriptor: dataops.Descriptor, Wire: dataops.Wire,
				Drivers: []string{"csv", "excel"}},
			{Descriptor: feature.Descriptor, Wire: feature.Wire},
			{Descriptor: uploadsec.Descriptor, Wire: uploadsec.Wire},
			{Descriptor: search.Descriptor, Wire: search.Wire},
			{Descriptor: retention.Descriptor, Wire: retention.Wire, Ungated: true},
			{Descriptor: apidocs.Descriptor, Wire: apidocs.Wire, Ungated: true},
			{Descriptor: grpcpkg.Descriptor, Wire: grpcpkg.Wire, Ungated: true},
			{Descriptor: ws.Descriptor, Wire: ws.Wire, Ungated: true},
		},
		Seed: profiles.StructuralSeed,
	}
}
