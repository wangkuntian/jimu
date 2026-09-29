package app

import (
	"fmt"
	"os"

	"jimu/internal/capabilities/access/domain"
	tenantDomain "jimu/internal/capabilities/tenant/domain"
	userdomain "jimu/internal/capabilities/user/domain"
	"jimu/internal/contract"
	"jimu/internal/kernel/access"
	"jimu/internal/kernel/tenant"

	"github.com/casbin/casbin/v3"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// RunSeed 插入初始数据
// 管理员密码从 ADMIN_PASSWORD 环境变量获取
// 权限点来自启用集各能力的 Descriptor.Permissions（能力自声明，未启用不种）
func RunSeed(db *gorm.DB, caps []contract.Descriptor) error {
	adminPassword := os.Getenv("ADMIN_PASSWORD")
	if adminPassword == "" {
		return fmt.Errorf("ADMIN_PASSWORD is required")
	}

	return db.Transaction(func(tx *gorm.DB) error {
		// 0. 确保默认租户存在（迁移 005 已写入；此处兜底，如跳过迁移直接 seed）
		defaultTenant := tenantDomain.Tenant{ID: tenant.DefaultTenantID, Code: "default", Name: "默认租户", Status: 1}
		if err := tx.Where("code = ?", defaultTenant.Code).FirstOrCreate(&defaultTenant).Error; err != nil {
			return fmt.Errorf("seed default tenant failed: %w", err)
		}

		// 1. 内置套餐示例（不自动分配给任何租户，需平台管理员显式分配才生效）
		freePlan := tenantDomain.Plan{Code: "free", Name: "免费版", MaxUsers: 10, MaxRoles: 5, MaxAPIKeys: 2}
		if err := tx.Where("code = ?", freePlan.Code).FirstOrCreate(&freePlan).Error; err != nil {
			return fmt.Errorf("seed free plan failed: %w", err)
		}

		// 2. 创建权限（聚合自启用集各能力的 Descriptor.Permissions）
		var permissions []domain.Permission
		for _, d := range caps {
			for _, p := range d.Permissions {
				permissions = append(permissions, domain.Permission{Name: p.Name, Resource: p.Resource, Action: p.Action})
			}
		}

		for i := range permissions {
			if err := tx.Where("resource = ? AND action = ?", permissions[i].Resource, permissions[i].Action).
				FirstOrCreate(&permissions[i]).Error; err != nil {
				return fmt.Errorf("seed permission failed: %w", err)
			}
		}

		// 3. 创建超级管理员角色（归属默认租户）
		adminRole := domain.Role{Name: "超级管理员", Description: "拥有所有权限", TenantID: tenant.DefaultTenantID}
		if err := tx.Where("name = ? AND tenant_id = ?", adminRole.Name, tenant.DefaultTenantID).FirstOrCreate(&adminRole).Error; err != nil {
			return fmt.Errorf("seed admin role failed: %w", err)
		}

		// 4. 为超级管理员分配所有权限
		for _, perm := range permissions {
			var count int64
			if err := tx.Table("role_permissions").Where("role_id = ? AND permission_id = ?", adminRole.ID, perm.ID).Count(&count).Error; err != nil {
				return fmt.Errorf("check role permission failed: %w", err)
			}
			if count == 0 {
				if err := tx.Exec("INSERT INTO role_permissions (role_id, permission_id) VALUES (?, ?)", adminRole.ID, perm.ID).Error; err != nil {
					return fmt.Errorf("assign permission failed: %w", err)
				}
			}
		}

		// 5. 创建默认管理员用户
		var adminUser userdomain.User
		result := tx.Where("username = ?", "admin").First(&adminUser)
		if result.Error == gorm.ErrRecordNotFound {
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
			if err != nil {
				return fmt.Errorf("hash password failed: %w", err)
			}
			adminUser = userdomain.User{
				Username: "admin",
				Password: string(hashedPassword),
				Status:   1,
				TenantID: tenant.DefaultTenantID,
			}
			if err := tx.Create(&adminUser).Error; err != nil {
				return fmt.Errorf("seed admin user failed: %w", err)
			}

			// 6. 为管理员分配超级管理员角色
			if err := tx.Exec("INSERT INTO user_roles (user_id, role_id) VALUES (?, ?)", adminUser.ID, adminRole.ID).Error; err != nil {
				return fmt.Errorf("assign admin role failed: %w", err)
			}
		}

		return nil
	})
}

// SeedCasbinPolicies 同步数据库中的角色-权限关系到 Casbin
// 应在 RunSeed 之后调用
func SeedCasbinPolicies(db *gorm.DB) error {
	enforcer, err := access.NewEnforcer(db)
	if err != nil {
		return fmt.Errorf("create enforcer: %w", err)
	}

	// 清空现有策略
	enforcer.ClearPolicy()

	// 从数据库加载角色-权限关系
	var policies []struct {
		Role     string `gorm:"column:role"`
		Resource string `gorm:"column:resource"`
		Action   string `gorm:"column:action"`
	}

	err = db.Table("roles").
		Select("roles.name AS role, permissions.resource, permissions.action").
		Joins("JOIN role_permissions rp ON rp.role_id = roles.id").
		Joins("JOIN permissions ON permissions.id = rp.permission_id").
		Scan(&policies).Error
	if err != nil {
		return fmt.Errorf("load role permissions: %w", err)
	}

	// 批量添加策略
	for _, p := range policies {
		_, err := enforcer.AddPolicy(p.Role, p.Resource, p.Action)
		if err != nil {
			return fmt.Errorf("add policy (%s, %s, %s): %w", p.Role, p.Resource, p.Action, err)
		}
	}

	return enforcer.SavePolicy()
}

// RunSeedWithCasbin 执行完整种子（含 Casbin 策略）
func RunSeedWithCasbin(db *gorm.DB, caps []contract.Descriptor) error {
	if err := RunSeed(db, caps); err != nil {
		return err
	}
	return SeedCasbinPolicies(db)
}

// 确保接口实现
var _ = func() *casbin.Enforcer { return nil }
