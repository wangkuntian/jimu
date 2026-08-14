package grpc

import (
	"context"

	userdomain "jimu/internal/modules/user/domain"
	"jimu/internal/platform/grpc/userinfopb"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

// userInfoService UserInfoService 的 gRPC 实现示例。
// 业务逻辑直连 GORM（示例），真实项目应注入 user 模块的 repository/service。
type userInfoService struct {
	userinfopb.UnimplementedUserInfoServiceServer
	db *gorm.DB
}

// NewUserInfoGRPCService 创建业务 gRPC 服务实现
func NewUserInfoGRPCService(db *gorm.DB) userinfopb.UserInfoServiceServer {
	return &userInfoService{db: db}
}

// GetUser 按 ID 查询用户。错误通过 status/codes 表达：
// 404 -> NotFound，内部错误 -> Internal。
func (s *userInfoService) GetUser(ctx context.Context, req *userinfopb.GetUserRequest) (*userinfopb.UserInfo, error) {
	if req == nil || req.GetUserId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	var u userdomain.User
	if err := s.db.WithContext(ctx).First(&u, req.GetUserId()).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, status.Errorf(codes.NotFound, "user %d not found", req.GetUserId())
		}
		return nil, status.Errorf(codes.Internal, "query user failed: %v", err)
	}

	return &userinfopb.UserInfo{
		UserId:    u.ID,
		Username:  u.Username,
		Status:    int32(u.Status),
		CreatedAt: u.CreatedAt.Format("2006-01-02 15:04:05"),
	}, nil
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

	var total int64
	if err := s.db.WithContext(ctx).Model(&userdomain.User{}).Count(&total).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "count users failed: %v", err)
	}

	var users []userdomain.User
	if err := s.db.WithContext(ctx).
		Offset((page - 1) * pageSize).Limit(pageSize).
		Order("id DESC").Find(&users).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "list users failed: %v", err)
	}

	resp := &userinfopb.ListUsersResponse{
		Total:    total,
		Page:     int32(page),
		PageSize: int32(pageSize),
		Users:    make([]*userinfopb.UserInfo, 0, len(users)),
	}
	for _, u := range users {
		resp.Users = append(resp.Users, &userinfopb.UserInfo{
			UserId:    u.ID,
			Username:  u.Username,
			Status:    int32(u.Status),
			CreatedAt: u.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return resp, nil
}
