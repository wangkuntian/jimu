// Package grpc 提供框架级 gRPC server 接入能力：生命周期管理、标准健康检查与反射。
// 业务服务通过 RegisterService 注册（参考 ping.go 的免 protoc ServiceDesc 写法）。
package grpc

import (
	"context"
	"fmt"
	"net"

	"jimu/internal/platform/logger"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

// Config gRPC server 配置
type Config struct {
	Enabled bool   `mapstructure:"enabled"` // 是否启用 gRPC server
	Host    string `mapstructure:"host"`    // 监听地址
	Port    int    `mapstructure:"port"`    // 监听端口
}

// Server 框架级 gRPC server：与 HTTP 双栈并存，默认注册健康检查（grpc_health_v1）与反射（grpcurl 可探）。
type Server struct {
	cfg      Config
	logger   *logger.Logger
	srv      *grpc.Server
	health   *health.Server
	listener net.Listener
}

// New 创建 gRPC server（不监听，Start 时才绑定端口）。
// logger 可为 nil（跳过日志输出）。
func New(cfg Config, log *logger.Logger) *Server {
	srv := grpc.NewServer()
	h := health.NewServer()
	healthpb.RegisterHealthServer(srv, h)
	reflection.Register(srv)
	RegisterPingServer(srv, &pingService{})
	return &Server{cfg: cfg, logger: log, srv: srv, health: h}
}

// RegisterService 注册 gRPC 服务（grpc.ServiceDesc + 实现），供业务模块接入。
func (s *Server) RegisterService(desc *grpc.ServiceDesc, impl interface{}) {
	s.srv.RegisterService(desc, impl)
}

// GRPCServer 暴露底层 server，供高级用法（拦截器、流控配置等）。
func (s *Server) GRPCServer() *grpc.Server {
	return s.srv
}

// Addr 返回实际监听地址（Port 为 0 时由系统分配，测试用）。
func (s *Server) Addr() string {
	if s.listener == nil {
		return ""
	}
	return s.listener.Addr().String()
}

// Start 实现 contract.Component：绑定端口并后台 serve。
func (s *Server) Start(ctx context.Context) error {
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
	var lc net.ListenConfig
	lis, err := lc.Listen(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("grpc listen %s: %w", addr, err)
	}
	s.listener = lis
	s.health.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	if s.logger != nil {
		s.logger.Info("grpc server listening", "addr", addr)
	}
	go func() {
		if err := s.srv.Serve(lis); err != nil {
			if s.logger != nil {
				s.logger.Error("grpc serve stopped", "error", err.Error())
			}
		}
	}()
	return nil
}

// Stop 实现 contract.Component：优雅停机（等在途请求完成）。
func (s *Server) Stop(_ context.Context) error {
	s.health.Shutdown()
	s.srv.GracefulStop()
	return nil
}
