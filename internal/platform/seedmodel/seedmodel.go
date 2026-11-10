// Package seedmodel 定义内核启动播种所需的最小持久化模型。
//
// 这些结构只承载表结构（字段、GORM tag、表名），不含仓储接口或业务方法。
// 它们让 internal/app 的 seed 不再 import 能力 domain 包，从而 0 能力项目也能编译。
//
// 字段与 tag 必须与对应能力 domain 逐字一致；漂移由 seedmodel_test.go 的结构断言拦截。
package seedmodel

import (
	"time"

	"jimu/internal/contract"

	"gorm.io/gorm"
)

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

type Role struct {
	ID          uint64             `gorm:"primaryKey" json:"id"`
	TenantID    uint64             `gorm:"column:tenant_id;default:0;uniqueIndex:idx_roles_tenant_name,priority:1" json:"tenant_id"`
	Name        string             `gorm:"size:64;not null;uniqueIndex:idx_roles_tenant_name,priority:2" json:"name"`
	Description string             `gorm:"size:255;default:''" json:"description"`
	Code        string             `gorm:"column:code;size:64;not null;default:'';index:idx_roles_code" json:"code"`
	Scope       contract.RoleScope `gorm:"size:16;not null;default:'tenant';index:idx_roles_scope" json:"scope"`
	IsSystem    bool               `gorm:"column:is_system;not null;default:false" json:"is_system"`
	Status      int8               `gorm:"default:1" json:"status"`
	Version     int64              `gorm:"not null;default:0" json:"-"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
	DeletedAt   gorm.DeletedAt     `gorm:"index" json:"-"`
}

func (Role) TableName() string { return "roles" }

type Tenant struct {
	ID        uint64         `gorm:"primaryKey" json:"id"`
	Code      string         `gorm:"size:64;uniqueIndex;not null" json:"code"`
	Name      string         `gorm:"size:128;not null" json:"name"`
	Status    int8           `gorm:"default:1" json:"status"`
	PlanID    uint64         `gorm:"not null;default:0" json:"plan_id"`
	Version   int64          `gorm:"not null;default:0" json:"-"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Tenant) TableName() string { return "tenants" }

type Plan struct {
	ID         uint64    `gorm:"primaryKey" json:"id"`
	Code       string    `gorm:"size:32;uniqueIndex;not null" json:"code"`
	Name       string    `gorm:"size:64;not null" json:"name"`
	MaxUsers   int       `gorm:"not null;default:0" json:"max_users"`
	MaxRoles   int       `gorm:"not null;default:0" json:"max_roles"`
	MaxAPIKeys int       `gorm:"not null;default:0" json:"max_api_keys"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (Plan) TableName() string { return "tenant_plans" }

type User struct {
	ID        uint64         `gorm:"primaryKey" json:"id"`
	Username  string         `gorm:"size:64;uniqueIndex;not null" json:"username"`
	Password  string         `gorm:"size:255;not null" json:"-"`
	Email     string         `gorm:"type:text" encryption:"true" json:"email"`
	EmailHash *string        `gorm:"size:64;uniqueIndex" blind:"email" json:"-"`
	Phone     string         `gorm:"type:text" encryption:"true" json:"phone"`
	PhoneHash *string        `gorm:"size:64;uniqueIndex" blind:"phone" json:"-"`
	Status    int8           `gorm:"default:1" json:"status"`
	TenantID  uint64         `gorm:"column:tenant_id;default:0;index" json:"tenant_id"`
	Version   int64          `gorm:"not null;default:0" json:"-"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (User) TableName() string { return "users" }
