package domain

import "context"

type UserRepository interface {
	FindByID(ctx context.Context, id uint64) (*User, error)
	FindByUsername(ctx context.Context, username string) (*User, error)
	FindByEmailHash(ctx context.Context, hash string) (*User, error)
	FindByPhoneHash(ctx context.Context, hash string) (*User, error)
	List(ctx context.Context, offset, limit int, sort, order string) ([]User, int64, error)
	Create(ctx context.Context, user *User) error
	Update(ctx context.Context, user *User) error
	UpdatePassword(ctx context.Context, id uint64, hashedPassword string) error
	UpdateTOTP(ctx context.Context, id uint64, secret string, enabled bool) error
	Delete(ctx context.Context, id uint64) error
}
