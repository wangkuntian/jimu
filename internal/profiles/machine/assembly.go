// Package machine 是无界面、服务间调用形态（设计 §2 ④）：catalog 取 user/access/apikey，
// 非 catalog 取 grpc（服务间入口）与 encryption（字段级加密）。没有 auth，因而不存在任何
// 登录/注册/会话端点；受保护路由由 apikey 经 X-API-Key + scope 承担（apikey.Wire 在
// auth 端口缺席时提供受保护中间件）。
package machine

import (
	"jimu/internal/assembly"
	accessmodule "jimu/internal/capabilities/access"
	"jimu/internal/capabilities/apikey"
	"jimu/internal/capabilities/encryption"
	grpcpkg "jimu/internal/capabilities/grpc"
	"jimu/internal/capabilities/user"
	"jimu/internal/profiles"
)

// Assembly 返回 machine 形态的装配清单。顺序即装配顺序：encryption 先提供 Cipher 端口，
// access 先于 user，apikey 读 auth 端口（缺席即由它提供受保护中间件），grpc 最后
// （经 user.info 端口注册 UserInfoService）。
//
// 已知限制：`/api/v1/admin/apikeys` 位于 `middleware.AdminAuth()` 之后，需要 machine
// 刻意排除的 JWT 链，因此该形态可启动但无法自助签发第一把 API Key —— 需要带外签发路径
// （P2.6 已提供：`PROFILE=machine make build-cli && ./bin/jimu-cli-machine apikey issue --name=first`，
// 见 README「形态（profile）」/ release note）。
func Assembly() assembly.Assembly {
	return assembly.Assembly{
		Name:    "machine",
		Version: profiles.Version,
		Capabilities: []assembly.Capability{
			{Descriptor: encryption.Descriptor, Wire: encryption.Wire, Ungated: true},
			{Descriptor: accessmodule.Descriptor, Wire: accessmodule.Wire},
			{Descriptor: user.Descriptor, Wire: user.Wire},
			{Descriptor: apikey.Descriptor, Wire: apikey.Wire},
			{Descriptor: grpcpkg.Descriptor, Wire: grpcpkg.Wire, Ungated: true},
		},
		Seed: profiles.StructuralSeed,
	}
}
