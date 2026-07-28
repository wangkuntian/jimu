package domain

import (
	"context"
	"time"

	"gorm.io/gorm"
)

type Role struct {
	ID          uint64         `gorm:"primaryKey" json:"id"`
	Name        string         `gorm:"size:64;uniqueIndex;not null" json:"name"`
	Description string         `gorm:"size:255;default:''" json:"description"`
	Status      int8           `gorm:"default:1" json:"status"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Role) TableName() string { return "roles" }

type RoleRepository interface {
	FindByID(ctx context.Context, id uint64) (*Role, error)
	FindAll(ctx context.Context) ([]Role, error)
	Create(ctx context.Context, role *Role) error
	Update(ctx context.Context, role *Role) error
	Delete(ctx context.Context, id uint64) error
	AssignPermissions(ctx context.Context, roleID uint64, permissionIDs []uint64) error
	GetPermissions(ctx context.Context, roleID uint64) ([]Permission, error)
}
