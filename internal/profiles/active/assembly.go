// Package active 是当前构建形态的唯一选择点：只 import 一个形态包，未选中的形态
// 不进 import 图 —— 层②的裁剪由它决定（设计 §6.3）。
//
// 提交态恒为 full：`go build ./cmd/server`、`go test ./...`、IDE 与 make swagger 都用它。
// 其它形态由 `tools/profileoverlay` 生成的 overlay 在构建期替换本文件（不改工作区）。
package active

import (
	"jimu/internal/assembly"
	"jimu/internal/profiles/full"
)

// Assembly 返回本次构建选中的形态。
func Assembly() assembly.Assembly { return full.Assembly() }
