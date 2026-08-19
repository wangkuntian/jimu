package e2e

import (
	"context"
	"testing"

	"jimu/internal/config"
	grpcpkg "jimu/internal/platform/grpc"
	"jimu/internal/platform/logger"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	reflectionpb "google.golang.org/grpc/reflection/grpc_reflection_v1"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// startTestGRPCServer 启动零端口 gRPC server（Ping + 健康检查 + 反射），返回已连接的客户端。
func startTestGRPCServer(t *testing.T) (*grpcpkg.Server, *grpc.ClientConn) {
	t.Helper()
	log := logger.New(config.LogConfig{Level: "warn", Format: "console", Output: "stdout"})
	s := grpcpkg.New(grpcpkg.Config{Host: "127.0.0.1", Port: 0}, log)
	require.NoError(t, s.Start(context.Background()))
	t.Cleanup(func() { _ = s.Stop(context.Background()) })

	conn, err := grpc.NewClient(s.Addr(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	return s, conn
}

// TestGRPCHealthCheck 验证 gRPC 健康检查协议：启动后返回 SERVING。
func TestGRPCHealthCheck(t *testing.T) {
	_, conn := startTestGRPCServer(t)

	hc := healthpb.NewHealthClient(conn)
	resp, err := hc.Check(context.Background(), &healthpb.HealthCheckRequest{})
	require.NoError(t, err)
	assert.Equal(t, healthpb.HealthCheckResponse_SERVING, resp.Status)
}

// TestGRPCPingEcho 验证 Ping 服务：echo 应答 + 空参数校验。
func TestGRPCPingEcho(t *testing.T) {
	_, conn := startTestGRPCServer(t)

	var out wrapperspb.StringValue
	err := conn.Invoke(context.Background(), "/jimu.v1.Ping/Ping", wrapperspb.String("hello"), &out)
	require.NoError(t, err)
	assert.Equal(t, "pong:hello", out.Value)

	// 空消息 → 参数校验
	err = conn.Invoke(context.Background(), "/jimu.v1.Ping/Ping", wrapperspb.String(""), &out)
	require.Error(t, err)
}

// TestGRPCReflection 验证 gRPC 反射：grpcurl 可探测服务列表。
func TestGRPCReflection(t *testing.T) {
	_, conn := startTestGRPCServer(t)

	ref := reflectionpb.NewServerReflectionClient(conn)
	stream, err := ref.ServerReflectionInfo(context.Background())
	require.NoError(t, err)
	require.NoError(t, stream.Send(&reflectionpb.ServerReflectionRequest{
		MessageRequest: &reflectionpb.ServerReflectionRequest_ListServices{ListServices: ""},
	}))

	resp, err := stream.Recv()
	require.NoError(t, err)

	var names []string
	for _, svc := range resp.GetListServicesResponse().GetService() {
		names = append(names, svc.GetName())
	}
	assert.Contains(t, names, "jimu.v1.Ping")
	assert.Contains(t, names, "grpc.health.v1.Health")
}
