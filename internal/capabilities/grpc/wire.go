package grpc

import (
	"fmt"

	"jimu/internal/assembly"
	"jimu/internal/capabilities/user"
	"jimu/internal/contract"
)

// Wire 装配 gRPC 接入能力：构造与 HTTP 双栈的 gRPC server，经 user.info 端口注册
// UserInfoService（user 未启用时跳过），cfg.GRPC.Enabled 时把 server 纳入应用生命周期。
func Wire(ctx *assembly.Context) (contract.Module, error) {
	cfg := ctx.Config()
	server, err := New(Config{
		Enabled:    cfg.GRPC.Enabled,
		Host:       cfg.GRPC.Host,
		Port:       cfg.GRPC.Port,
		TimeoutSec: cfg.GRPC.TimeoutSec,
		TLS:        cfg.GRPC.TLS,
	}, ctx.Logger(), ctx.Reporter())
	if err != nil {
		return nil, fmt.Errorf("init grpc server: %w", err)
	}
	// 业务示例：注册 UserInfoService，用户数据经 contract.UserinfoSource 端口读取
	// （user 能力提供适配实现，grpc 能力不直接依赖 user/domain）
	if source, ok := ctx.Port(user.UserinfoPortName).(contract.UserinfoSource); ok && source != nil {
		server.RegisterUserInfoService(source)
	}
	if cfg.GRPC.Enabled {
		ctx.RegisterComponent(server)
	}
	return nil, nil
}
