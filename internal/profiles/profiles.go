// Package profiles 提供各形态（profile）共享的装配构件：构建版本默认值与结构性种子。
//
// 形态分两层：internal/profiles/<name> 声明该形态的能力清单（package 非 main），
// 唯一入口 cmd/server 经选点包 internal/profiles/active 取当前形态（提交态默认 full，
// 构建期由 tools/profileoverlay 的 overlay 切换）；共享件只放在本包，避免各形态互相 import。
package profiles

// Version 是形态清单默认填入的构建版本；唯一入口可经 ldflags 注入后覆盖
// （cmd/server 沿用 `-X main.version`，Makefile/Dockerfile 不变）。
var Version = "dev"
