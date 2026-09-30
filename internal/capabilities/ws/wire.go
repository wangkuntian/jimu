package ws

import (
	"fmt"

	"jimu/internal/assembly"
	"jimu/internal/contract"
)

const PortName = "ws"

// Wire 装配 WebSocket 能力，提供管理端口并把 hub 纳入应用生命周期。
func Wire(ctx *assembly.Context) (contract.Module, error) {
	port := newAdminWebSocket()
	if err := ctx.Provide(PortName, port); err != nil {
		return nil, fmt.Errorf("provide ws port: %w", err)
	}
	ctx.RegisterComponent(port)
	return nil, nil
}
