package grpc

import (
	"context"
	"net"
	"testing"

	userdomain "jimu/internal/modules/user/domain"
	"jimu/internal/platform/grpc/userinfopb"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTestGRPCService(t *testing.T) userinfopb.UserInfoServiceClient {
	t.Helper()
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, gdb.AutoMigrate(&userdomain.User{}))

	// 种子数据
	require.NoError(t, gdb.Create(&userdomain.User{ID: 1, Username: "alice", Status: 1}).Error)
	require.NoError(t, gdb.Create(&userdomain.User{ID: 2, Username: "bob", Status: 1}).Error)

	// 用随机端口模拟 gRPC 连接，避免真实端口冲突
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	srv := grpc.NewServer()
	userinfopb.RegisterUserInfoServiceServer(srv, NewUserInfoGRPCService(gdb))
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	return userinfopb.NewUserInfoServiceClient(conn)
}

func TestUserInfoService_GetUser(t *testing.T) {
	client := newTestGRPCService(t)

	// 正常查询
	u, err := client.GetUser(context.Background(), &userinfopb.GetUserRequest{UserId: 1})
	require.NoError(t, err)
	require.Equal(t, "alice", u.Username)
	require.Equal(t, int32(1), u.Status)

	// 不存在 → NotFound
	_, err = client.GetUser(context.Background(), &userinfopb.GetUserRequest{UserId: 999})
	require.Error(t, err)
	require.Equal(t, codes.NotFound, status.Code(err))

	// 参数非法 → InvalidArgument
	_, err = client.GetUser(context.Background(), &userinfopb.GetUserRequest{})
	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestUserInfoService_ListUsers(t *testing.T) {
	client := newTestGRPCService(t)

	resp, err := client.ListUsers(context.Background(), &userinfopb.ListUsersRequest{Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Equal(t, int64(2), resp.Total)
	require.Len(t, resp.Users, 2)
	require.Equal(t, int32(1), resp.Page)
	require.Equal(t, int32(10), resp.PageSize)

	// 非法分页参数回落默认值
	resp, err = client.ListUsers(context.Background(), &userinfopb.ListUsersRequest{})
	require.NoError(t, err)
	require.Equal(t, int32(1), resp.Page)
	require.Equal(t, int32(20), resp.PageSize)
}
