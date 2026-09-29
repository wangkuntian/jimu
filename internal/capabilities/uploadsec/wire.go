package uploadsec

import (
	"jimu/internal/assembly"
	"jimu/internal/capabilities/storage"
	"jimu/internal/contract"
)

// Wire 装配文件上传安全能力：消费 storage 端口取得存储实现，按 upload 段构造病毒
// 扫描器（未启用 ClamAV 时为 nil，不扫描），返回上传管理模块。
func Wire(ctx *assembly.Context) (contract.Module, error) {
	st, _ := ctx.Port(storage.PortName).(storage.Storage)
	var scanner Scanner
	if cfg := assembly.MustSection[*Config](ctx, ConfigKey); cfg != nil {
		scanner = cfg.Scanner()
	}
	return New(st, scanner), nil
}
