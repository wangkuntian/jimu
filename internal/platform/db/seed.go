package db

import (
	"fmt"
	"os"

	"jimu/internal/modules/role/domain"
	tenantDomain "jimu/internal/modules/tenant/domain"
	userdomain "jimu/internal/modules/user/domain"
	"jimu/internal/platform/auth"
	"jimu/internal/platform/tenant"

	"github.com/casbin/casbin/v3"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// RunSeed 插入初始数据
// 管理员密码从 ADMIN_PASSWORD 环境变量获取
func RunSeed(db *gorm.DB) error {
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

		// 2. 创建基础权限
		permissions := basePermissions()

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
	enforcer, err := auth.NewEnforcer(db)
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
func RunSeedWithCasbin(db *gorm.DB) error {
	if err := RunSeed(db); err != nil {
		return err
	}
	return SeedCasbinPolicies(db)
}

// 确保接口实现
var _ = func() *casbin.Enforcer { return nil }

func basePermissions() []domain.Permission {
	return []domain.Permission{
		{Name: "用户列表", Resource: "/api/v1/users", Action: "GET"},
		{Name: "用户创建", Resource: "/api/v1/users", Action: "POST"},
		{Name: "用户详情", Resource: "/api/v1/users/*", Action: "GET"},
		{Name: "用户修改", Resource: "/api/v1/users/*", Action: "PUT"},
		{Name: "用户删除", Resource: "/api/v1/users/*", Action: "DELETE"},
		{Name: "角色列表", Resource: "/api/v1/roles", Action: "GET"},
		{Name: "角色创建", Resource: "/api/v1/roles", Action: "POST"},
		{Name: "角色详情", Resource: "/api/v1/roles/*", Action: "GET"},
		{Name: "角色修改", Resource: "/api/v1/roles/*", Action: "PUT"},
		{Name: "角色删除", Resource: "/api/v1/roles/*", Action: "DELETE"},
		{Name: "角色分配权限", Resource: "/api/v1/roles/*/permissions", Action: "POST"},
		{Name: "权限列表", Resource: "/api/v1/permissions", Action: "GET"},
		{Name: "权限创建", Resource: "/api/v1/permissions", Action: "POST"},
		{Name: "权限详情", Resource: "/api/v1/permissions/*", Action: "GET"},
		{Name: "权限修改", Resource: "/api/v1/permissions/*", Action: "PUT"},
		{Name: "权限删除", Resource: "/api/v1/permissions/*", Action: "DELETE"},
		{Name: "审计列表", Resource: "/api/v1/audits", Action: "GET"},
		{Name: "审计详情", Resource: "/api/v1/audits/*", Action: "GET"},
		{Name: "租户列表", Resource: "/api/v1/tenants", Action: "GET"},
		{Name: "租户创建", Resource: "/api/v1/tenants", Action: "POST"},
		{Name: "租户详情", Resource: "/api/v1/tenants/*", Action: "GET"},
		{Name: "租户修改", Resource: "/api/v1/tenants/*", Action: "PUT"},
		{Name: "租户删除", Resource: "/api/v1/tenants/*", Action: "DELETE"},
		// 租户运营：套餐定义（用量查询与套餐分配分别由「租户详情」「租户修改」通配覆盖）
		{Name: "套餐列表", Resource: "/api/v1/tenant-plans", Action: "GET"},
		{Name: "套餐创建", Resource: "/api/v1/tenant-plans", Action: "POST"},
		{Name: "套餐修改", Resource: "/api/v1/tenant-plans/*", Action: "PUT"},
		{Name: "套餐删除", Resource: "/api/v1/tenant-plans/*", Action: "DELETE"},
		// 管理后台端点（/api/v1/admin/* 由 keyMatch 通配覆盖全部管理 API）
		{Name: "管理后台读取", Resource: "/api/v1/admin/*", Action: "GET"},
		{Name: "管理后台写入", Resource: "/api/v1/admin/*", Action: "POST"},
		{Name: "管理后台修改", Resource: "/api/v1/admin/*", Action: "PUT"},
		{Name: "管理后台删除", Resource: "/api/v1/admin/*", Action: "DELETE"},
	}
}
