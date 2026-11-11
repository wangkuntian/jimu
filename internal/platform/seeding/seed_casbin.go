package seeding

import (
	"fmt"

	"jimu/internal/platform/authz"

	"gorm.io/gorm"
)

// 独立成文件：生成器按能力集裁剪时，未启用 access 的项目整份排除，
// 从而不把 Casbin 与 gorm-adapter 编进二进制。seed.go 里有同名变量的空实现，
// 本文件的 init() 会把它替换成真实实现。
func init() { seedCasbinPolicies = syncCasbinPolicies }

func syncCasbinPolicies(db *gorm.DB) error {
	enforcer, err := authz.NewEnforcer(db)
	if err != nil {
		return fmt.Errorf("create enforcer: %w", err)
	}

	// 清空现有策略
	enforcer.ClearPolicy()

	// 从数据库加载角色-权限关系
	var policies []struct {
		RoleID   uint64 `gorm:"column:role_id"`
		Resource string `gorm:"column:resource"`
		Action   string `gorm:"column:action"`
	}

	err = db.Table("roles").
		Select("roles.id AS role_id, permissions.resource, permissions.action").
		Joins("JOIN role_permissions rp ON rp.role_id = roles.id").
		Joins("JOIN permissions ON permissions.id = rp.permission_id").
		Scan(&policies).Error
	if err != nil {
		return fmt.Errorf("load role permissions: %w", err)
	}

	// 批量添加策略
	for _, p := range policies {
		_, err := enforcer.AddPolicy(authz.RoleSubject(p.RoleID), p.Resource, p.Action)
		if err != nil {
			return fmt.Errorf("add policy (%d, %s, %s): %w", p.RoleID, p.Resource, p.Action, err)
		}
	}

	return enforcer.SavePolicy()
}
