package domain

import "context"

// UserListFilter 用户列表筛选。TenantID 由服务层按身份范围收敛后传入。
type UserListFilter struct {
	TenantID uint64
	Username string
	Status   *int8
}

type UserRepository interface {
	FindByID(ctx context.Context, id uint64) (*User, error)
	FindByUsername(ctx context.Context, username string) (*User, error)
	FindByEmailHash(ctx context.Context, hash string) (*User, error)
	FindByPhoneHash(ctx context.Context, hash string) (*User, error)
	List(ctx context.Context, filter UserListFilter, offset, limit int, sort, order string) ([]User, int64, error)
	Create(ctx context.Context, user *User) error
	Update(ctx context.Context, user *User) error
	UpdatePassword(ctx context.Context, id uint64, hashedPassword string) error
	Delete(ctx context.Context, id uint64) error
}
