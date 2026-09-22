// Package catalog 维护全仓库唯一的能力清单。
//
// 解析算法（Resolve/ValidateDeclarations/Degraded）已迁至 internal/assembly，
// 本包只提供全量清单 All()/Names()，并保留同名薄封装以免破坏既有调用点。
//
// 新增能力：在 entries 中追加一行（位置必须在它的依赖之后）。
// 删除能力：删掉该行与对应目录，其余代码无需改动 —— 这是"可插拔"的中心点。
package catalog

import (
	"jimu/internal/assembly"
	accessmodule "jimu/internal/capabilities/access"
	"jimu/internal/capabilities/apikey"
	auditmodule "jimu/internal/capabilities/audit"
	authmodule "jimu/internal/capabilities/auth"
	"jimu/internal/capabilities/breach"
	"jimu/internal/capabilities/captcha"
	consolemodule "jimu/internal/capabilities/console"
	"jimu/internal/capabilities/dataops"
	"jimu/internal/capabilities/feature"
	mfamodule "jimu/internal/capabilities/mfa"
	oauthmodule "jimu/internal/capabilities/oauth"
	"jimu/internal/capabilities/outbox"
	passkeymodule "jimu/internal/capabilities/passkey"
	"jimu/internal/capabilities/queue"
	"jimu/internal/capabilities/search"
	tenantmodule "jimu/internal/capabilities/tenant"
	"jimu/internal/capabilities/uploadsec"
	"jimu/internal/capabilities/user"
	"jimu/internal/contract"
)

// entries 是唯一的能力清单，顺序即默认启用顺序（同时是依赖拓扑序）。
// 尾部基础设施能力（breach/captcha 等）无 Requires；breach 无 Module 实例，
// 仅携带声明与端口实现；captcha 本轮起有实例并自挂公开路由。
var entries = []contract.Descriptor{
	user.Descriptor,
	accessmodule.Descriptor,
	tenantmodule.Descriptor,
	mfamodule.Descriptor,
	authmodule.Descriptor,
	passkeymodule.Descriptor,
	auditmodule.Descriptor,
	consolemodule.Descriptor,
	oauthmodule.Descriptor,
	apikey.Descriptor,
	queue.Descriptor,
	outbox.Descriptor,
	dataops.Descriptor,
	search.Descriptor,
	captcha.Descriptor,
	feature.Descriptor,
	uploadsec.Descriptor,
	breach.Descriptor,
}

// ValidateDeclarations 校验全量清单的声明结构（薄封装，算法见 internal/assembly）。
func ValidateDeclarations() error {
	return assembly.ValidateDeclarations(entries)
}

// All 返回清单中全部能力的深拷贝（含 Requires/SoftRequires/Owns/Permissions/Configs），
// 调用方修改不影响清单。
func All() []contract.Descriptor {
	out := make([]contract.Descriptor, len(entries))
	for i, d := range entries {
		out[i] = d
		out[i].Requires = append([]string(nil), d.Requires...)
		out[i].SoftRequires = append([]string(nil), d.SoftRequires...)
		out[i].Owns = append([]string(nil), d.Owns...)
		out[i].Permissions = append([]contract.Permission(nil), d.Permissions...)
		out[i].Configs = append([]contract.ConfigSpec(nil), d.Configs...)
	}
	return out
}

// Names 返回清单中的能力名，按清单顺序。
func Names() []string {
	out := make([]string, 0, len(entries))
	for _, d := range entries {
		out = append(out, d.Name)
	}
	return out
}

// Degradation 是 assembly.Degradation 的别名，保留能力清单侧的既有引用。
type Degradation = assembly.Degradation

// Degraded 返回已解析启用集里被声明但缺失的软依赖（薄封装，算法见 internal/assembly）。
func Degraded(caps []contract.Descriptor) []Degradation {
	return assembly.Degraded(caps)
}

// Resolve 解析全量清单上的启用集（薄封装，算法见 internal/assembly）。
func Resolve(enabled []string) ([]contract.Descriptor, error) {
	return assembly.Resolve(entries, enabled)
}
