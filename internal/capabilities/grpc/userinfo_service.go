package grpc

import (
	"context"
	"errors"

	"jimu/internal/capabilities/grpc/userinfopb"
	"jimu/internal/contract"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// userInfoService UserInfoService 的 gRPC 实现。
// 用户数据经 contract.UserinfoSource 端口读取（user 能力提供实现），
// 错误映射：contract.ErrNotFound -> NotFound，其余 -> Internal。
type userInfoService struct {
	userinfopb.UnimplementedUserInfoServiceServer
	source contract.UserinfoSource
}

// NewUserInfoGRPCService 创建业务 gRPC 服务实现
func NewUserInfoGRPCService(source contract.UserinfoSource) userinfopb.UserInfoServiceServer {
	return &userInfoService{source: source}
}

// toUserInfo 端口视图转 proto 响应（CreatedAt 格式为原实现固定的 "2006-01-02 15:04:05"）
func toUserInfo(u *contract.Userinfo) *userinfopb.UserInfo {
	return &userinfopb.UserInfo{
		UserId:    u.ID,
		Username:  u.Username,
		Status:    int32(u.Status),
		CreatedAt: u.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}

// GetUser 按 ID 查询用户。错误通过 status/codes 表达：
// 404 -> NotFound，内部错误 -> Internal。
func (s *userInfoService) GetUser(ctx context.Context, req *userinfopb.GetUserRequest) (*userinfopb.UserInfo, error) {
	if req == nil || req.GetUserId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	u, err := s.source.GetByID(ctx, req.GetUserId())
	if err != nil {
		if errors.Is(err, contract.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "user %d not found", req.GetUserId())
		}
		return nil, status.Errorf(codes.Internal, "query user failed: %v", err)
	}

	return toUserInfo(u), nil
}

// ListUsers 分页查询用户。
func (s *userInfoService) ListUsers(ctx context.Context, req *userinfopb.ListUsersRequest) (*userinfopb.ListUsersResponse, error) {
	page := int(req.GetPage())
	if page <= 0 {
		page = 1
	}
	pageSize := int(req.GetPageSize())
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	users, total, err := s.source.List(ctx, page, pageSize)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list users failed: %v", err)
	}

	resp := &userinfopb.ListUsersResponse{
		Total:    total,
		Page:     int32(page),
		PageSize: int32(pageSize),
		Users:    make([]*userinfopb.UserInfo, 0, len(users)),
	}
	for i := range users {
		resp.Users = append(resp.Users, toUserInfo(&users[i]))
	}
	return resp, nil
}
