package domain

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

var ErrPermissionNotFound = errors.New("permission not found")

type Permission struct {
	ID        uint64         `gorm:"primaryKey" json:"id"`
	Name      string         `gorm:"size:128;not null" json:"name"`
	Resource  string         `gorm:"size:64;not null;uniqueIndex:uk_resource_action,priority:1" json:"resource"`
	Action    string         `gorm:"size:32;not null;uniqueIndex:uk_resource_action,priority:2" json:"action"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Permission) TableName() string { return "permissions" }

type PermissionRepository interface {
	FindByID(ctx context.Context, id uint64) (*Permission, error)
	List(ctx context.Context, offset, limit int, sort, order string) ([]Permission, int64, error)
	Create(ctx context.Context, perm *Permission) error
	Update(ctx context.Context, perm *Permission) error
	Delete(ctx context.Context, id uint64) error
}
