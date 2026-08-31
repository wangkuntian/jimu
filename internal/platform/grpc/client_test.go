package grpc

import (
	"context"
	"net"
	"testing"
	"time"

	userdomain "jimu/internal/modules/user/domain"
	"jimu/internal/platform/grpc/userinfopb"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

// startEchoServer 用真实 userInfoService（内存 sqlite）起测试 gRPC server。
func startEchoServer(t *testing.T, withUser bool) string {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&userdomain.User{}))
	if withUser {
		require.NoError(t, db.Create(&userdomain.User{Username: "alice", Password: "hash", Status: 1}).Error)
	}

	srv := grpc.NewServer()
	userinfopb.RegisterUserInfoServiceServer(srv, NewUserInfoGRPCService(db))
	lis := startMemoryListener(t)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)
	return lis.Addr().String()
}

// startMemoryListener 分配随机可用端口监听（测试用，非 bufconn，便于 client 走真实拨号路径）。
func startMemoryListener(t *testing.T) net.Listener {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	return lis
}

func TestNewClient_RequiresAddress(t *testing.T) {
	_, err := NewClient(ClientConfig{})
	require.Error(t, err)
}

func TestClient_Invoke(t *testing.T) {
	addr := startEchoServer(t, true)
	client, err := NewClient(ClientConfig{Address: addr, CallTimeout: 3 * time.Second, MaxRetries: 0})
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })

	svc := userinfopb.NewUserInfoServiceClient(client.Conn())
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	resp, err := svc.GetUser(ctx, &userinfopb.GetUserRequest{UserId: 1})
	require.NoError(t, err)
	assert.Equal(t, "alice", resp.Username)
}

func TestClient_NonRetryableInvalidArgument(t *testing.T) {
	// InvalidArgument 不可重试：客户端调用应直接失败不重试
	addr := startEchoServer(t, false)
	client, err := NewClient(ClientConfig{Address: addr, CallTimeout: 3 * time.Second, MaxRetries: 3, RetryInterval: 5 * time.Millisecond})
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })

	svcClient := userinfopb.NewUserInfoServiceClient(client.Conn())
	_, err = svcClient.GetUser(context.Background(), &userinfopb.GetUserRequest{UserId: 0})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestClient_NotFound(t *testing.T) {
	addr := startEchoServer(t, false)
	client, err := NewClient(ClientConfig{Address: addr, CallTimeout: 3 * time.Second, MaxRetries: 0})
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })

	svcClient := userinfopb.NewUserInfoServiceClient(client.Conn())
	_, err = svcClient.GetUser(context.Background(), &userinfopb.GetUserRequest{UserId: 999})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

// TestClient_RetryOnUnavailable 用不存在的端口验证重试路径（Unavailable 可重试，
// 最终返回错误而非 panic）。仅验证拦截器链无崩溃。
func TestClient_RetryOnUnavailable(t *testing.T) {
	client, err := NewClient(ClientConfig{Address: "127.0.0.1:1", CallTimeout: 1 * time.Second, MaxRetries: 1, RetryInterval: 5 * time.Millisecond})
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })

	svcClient := userinfopb.NewUserInfoServiceClient(client.Conn())
	_, err = svcClient.GetUser(context.Background(), &userinfopb.GetUserRequest{UserId: 1})
	require.Error(t, err)
	// 端口 1 通常不可达，返回 Unavailable 或 DeadlineExceeded
	assert.Contains(t, []codes.Code{codes.Unavailable, codes.DeadlineExceeded}, status.Code(err))
}

func TestClient_InvokeAndState(t *testing.T) {
	addr := startEchoServer(t, true)
	client, err := NewClient(ClientConfig{Address: addr, CallTimeout: 3 * time.Second, MaxRetries: 0, ServiceName: "test"})
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })

	// Invoke 动态调用路径（proto package jimu.v1）
	var resp userinfopb.UserInfo
	err = client.Invoke(context.Background(), "/jimu.v1.UserInfoService/GetUser", &userinfopb.GetUserRequest{UserId: 1}, &resp)
	require.NoError(t, err)
	assert.Equal(t, "alice", resp.Username)

	// State 返回连接状态且不崩溃（在 Close 前）
	_ = client.State()
	require.NoError(t, client.Close())
	// 重复 Close 幂等
	require.NoError(t, client.Close())
}

func TestDefaultClientConfig(t *testing.T) {
	cfg := DefaultClientConfig()
	if cfg.DialTimeout != 10*time.Second || cfg.CallTimeout != 10*time.Second {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
	if cfg.MaxRetries != 2 || cfg.RetryInterval != 200*time.Millisecond {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
}
