package storage

import (
	"fmt"

	"jimu/internal/assembly"
	"jimu/internal/contract"
)

// Wire 装配文件存储能力：解码 storage 段并构造存储实例，把 Storage 暴露为端口供
// uploadsec 消费。storage 是非 catalog 条目，其段由组合方无条件加载（不随启用集裁剪）。
func Wire(ctx *assembly.Context) (contract.Module, error) {
	cfg, err := Load(ctx.Sections())
	if err != nil {
		return nil, fmt.Errorf("init storage: %w", err)
	}
	svc, err := New(*cfg)
	if err != nil {
		return nil, fmt.Errorf("init storage: %w", err)
	}
	if err := ctx.Provide(PortName, svc); err != nil {
		return nil, fmt.Errorf("provide storage port: %w", err)
	}
	return nil, nil
}
