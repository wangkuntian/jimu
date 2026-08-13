package grpc

import (
	"context"
	"testing"

	"jimu/internal/config"
	"jimu/internal/platform/logger"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	reflectionpb "google.golang.org/grpc/reflection/grpc_reflection_v1"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func newTestLogger() *logger.Logger {
	return logger.New(config.LogConfig{Level: "warn", Format: "console", Output: "stdout"})
}

func startTestServer(t *testing.T) *Server {
	t.Helper()
	s := New(Config{Host: "127.0.0.1", Port: 0}, newTestLogger())
	require.NoError(t, s.Start(context.Background()))
	t.Cleanup(func() { _ = s.Stop(context.Background()) })
	return s
}

func dial(t *testing.T, s *Server) *grpc.ClientConn {
	t.Helper()
	conn, err := grpc.NewClient(s.Addr(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func TestPingEcho(t *testing.T) {
	s := startTestServer(t)
	conn := dial(t, s)

	var out wrapperspb.StringValue
	err := conn.Invoke(context.Background(), "/jimu.v1.Ping/Ping", wrapperspb.String("hi"), &out)
	require.NoError(t, err)
	assert.Equal(t, "pong:hi", out.Value)

	// 空消息走校验错误路径
	err = conn.Invoke(context.Background(), "/jimu.v1.Ping/Ping", wrapperspb.String(""), &out)
	require.Error(t, err)
}

func TestHealthServing(t *testing.T) {
	s := startTestServer(t)
	conn := dial(t, s)

	hc := healthpb.NewHealthClient(conn)
	resp, err := hc.Check(context.Background(), &healthpb.HealthCheckRequest{})
	require.NoError(t, err)
	assert.Equal(t, healthpb.HealthCheckResponse_SERVING, resp.Status)
}

func TestReflectionListsPingService(t *testing.T) {
	s := startTestServer(t)
	conn := dial(t, s)

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

func TestAddrBeforeStartIsEmpty(t *testing.T) {
	s := New(Config{Host: "127.0.0.1", Port: 9091}, newTestLogger())
	assert.Empty(t, s.Addr())
}
