package user

import (
	"context"
	"errors"

	"jimu/internal/capabilities/user/domain"
	"jimu/internal/contract"

	"gorm.io/gorm"
)

// userinfoAdapter contract.UserinfoSource 的薄适配：读取 user 仓储，
// 映射领域实体为端口视图。平台级视角（不过滤租户），与原 gRPC 直查语义一致。
type userinfoAdapter struct {
	repo domain.UserRepository
}

// NewUserinfoSource 构造 contract.UserinfoSource 端口实现。
func NewUserinfoSource(repo domain.UserRepository) contract.UserinfoSource {
	return &userinfoAdapter{repo: repo}
}

func (a *userinfoAdapter) GetByID(ctx context.Context, id uint64) (*contract.Userinfo, error) {
	u, err := a.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, contract.ErrNotFound
		}
		return nil, err
	}
	return toUserinfo(u), nil
}

func (a *userinfoAdapter) List(ctx context.Context, page, pageSize int) ([]contract.Userinfo, int64, error) {
	users, total, err := a.repo.List(ctx, 0, (page-1)*pageSize, pageSize, "id", "desc")
	if err != nil {
		return nil, 0, err
	}
	out := make([]contract.Userinfo, 0, len(users))
	for i := range users {
		out = append(out, *toUserinfo(&users[i]))
	}
	return out, total, nil
}

func toUserinfo(u *domain.User) *contract.Userinfo {
	return &contract.Userinfo{
		ID:        u.ID,
		Username:  u.Username,
		Status:    u.Status,
		CreatedAt: u.CreatedAt,
	}
}
