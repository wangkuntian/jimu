package grpc

import (
	"context"
	"crypto/tls"
	"testing"
	"time"

	"jimu/internal/config"
	"jimu/internal/platform/logger"
	"jimu/internal/platform/tlsconf"
	"jimu/internal/shared/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
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
	s, err := New(Config{Host: "127.0.0.1", Port: 0}, newTestLogger())
	require.NoError(t, err)
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
	s, err := New(Config{Host: "127.0.0.1", Port: 9091}, newTestLogger())
	require.NoError(t, err)
	assert.Empty(t, s.Addr())
}

// TestGRPCServerMTLS 验证 gRPC 侧 mTLS：带客户端证书可调用，缺失时握手失败。
func TestGRPCServerMTLS(t *testing.T) {
	material := testutil.NewTLSMaterial(t)
	s, err := New(Config{
		Host: "127.0.0.1",
		Port: 0,
		TLS: config.TLSConfig{
			Enabled:      true,
			CertFile:     material.ServerCertFile,
			KeyFile:      material.ServerKeyFile,
			ClientCAFile: material.ClientCAFile,
		},
	}, newTestLogger())
	require.NoError(t, err)
	require.NoError(t, s.Start(context.Background()))
	t.Cleanup(func() { _ = s.Stop(context.Background()) })

	caPool, err := tlsconf.LoadCAPool(material.ClientCAFile)
	require.NoError(t, err)
	clientCert, err := tls.LoadX509KeyPair(material.ClientCertFile, material.ClientKeyFile)
	require.NoError(t, err)

	// 带客户端证书：健康检查成功
	withCert := credentials.NewTLS(&tls.Config{
		RootCAs:      caPool,
		Certificates: []tls.Certificate{clientCert},
		MinVersion:   tls.VersionTLS12,
	})
	conn, err := grpc.NewClient(s.Addr(), grpc.WithTransportCredentials(withCert))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = healthpb.NewHealthClient(conn).Check(ctx, &healthpb.HealthCheckRequest{})
	assert.NoError(t, err, "携带客户端证书应能调用")

	// 不带客户端证书：握手被拒
	noCert := credentials.NewTLS(&tls.Config{RootCAs: caPool, MinVersion: tls.VersionTLS12})
	connNoCert, err := grpc.NewClient(s.Addr(), grpc.WithTransportCredentials(noCert))
	require.NoError(t, err)
	t.Cleanup(func() { _ = connNoCert.Close() })

	ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()
	_, err = healthpb.NewHealthClient(connNoCert).Check(ctx2, &healthpb.HealthCheckRequest{})
	assert.Error(t, err, "缺失客户端证书应调用失败")
}
