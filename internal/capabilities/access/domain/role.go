package domain

import (
	"context"
	"errors"
	"time"

	"jimu/internal/contract"

	"gorm.io/gorm"
)

var ErrRoleInUse = errors.New("role is assigned to users")

type Role struct {
	ID          uint64             `gorm:"primaryKey" json:"id"`
	TenantID    uint64             `gorm:"column:tenant_id;default:0;uniqueIndex:idx_roles_tenant_name,priority:1" json:"tenant_id"` // 所属租户 ID（0=未归属）
	Name        string             `gorm:"size:64;not null;uniqueIndex:idx_roles_tenant_name,priority:2" json:"name"`                // 租户内唯一
	Description string             `gorm:"size:255;default:''" json:"description"`
	Code        string             `gorm:"column:code;size:64;not null;default:'';index:idx_roles_code" json:"code"`
	Scope       contract.RoleScope `gorm:"size:16;not null;default:'tenant';index:idx_roles_scope" json:"scope"`
	IsSystem    bool               `gorm:"column:is_system;not null;default:false" json:"is_system"`
	Status      int8               `gorm:"default:1" json:"status"`
	Version     int64              `gorm:"not null;default:0" json:"-"` // 乐观锁版本号（每次更新自增）
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
	DeletedAt   gorm.DeletedAt     `gorm:"index" json:"-"`
}

func (Role) TableName() string { return "roles" }

// RoleListFilter 角色列表筛选条件；零值表示平台级不过滤。
type RoleListFilter struct {
	TenantID uint64
	Keyword  string
	Status   *int8
}

type RoleRepository interface {
	FindByID(ctx context.Context, id uint64) (*Role, error)
	List(ctx context.Context, filter RoleListFilter, offset, limit int, sort, order string) ([]Role, int64, error)
	ListOptions(ctx context.Context, tenantID uint64) ([]Role, error)
	FindPermissionsByIDs(ctx context.Context, ids []uint64) ([]Permission, error)
	Create(ctx context.Context, role *Role) error
	Update(ctx context.Context, role *Role) error
	Delete(ctx context.Context, id uint64) error
	AssignPermissions(ctx context.Context, roleID uint64, permissionIDs []uint64) error
	GetPermissions(ctx context.Context, roleID uint64) ([]Permission, error)
}
