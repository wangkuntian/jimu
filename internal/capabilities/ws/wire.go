package ws

import (
	"jimu/internal/assembly"
	"jimu/internal/contract"
)

// Wire 装配 WebSocket 能力：ws 是供 console 等能力复用的客户端 Hub 库，装配期无自建
// 实例、无端口、无 Module；仅作为形态条目参与启用集与能力报告。
func Wire(*assembly.Context) (contract.Module, error) { return nil, nil }
