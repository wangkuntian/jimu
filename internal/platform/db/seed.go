package db

import (
	"fmt"
	"os"

	"jimu/internal/modules/role/domain"
	userdomain "jimu/internal/modules/user/domain"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// RunSeed 插入初始数据
// 管理员密码优先从 SEED_ADMIN_PASSWORD 环境变量获取，默认 admin123
func RunSeed(db *gorm.DB) error {
	adminPassword := os.Getenv("SEED_ADMIN_PASSWORD")
	if adminPassword == "" {
		adminPassword = "admin123"
	}

	return db.Transaction(func(tx *gorm.DB) error {
		// 1. 创建基础权限
		permissions := []domain.Permission{
			{Name: "用户查看", Resource: "/api/v1/users", Action: "GET"},
			{Name: "用户创建", Resource: "/api/v1/users", Action: "POST"},
			{Name: "用户修改", Resource: "/api/v1/users/*", Action: "PUT"},
			{Name: "用户删除", Resource: "/api/v1/users/*", Action: "DELETE"},
			{Name: "角色查看", Resource: "/api/v1/roles", Action: "GET"},
			{Name: "角色创建", Resource: "/api/v1/roles", Action: "POST"},
			{Name: "角色修改", Resource: "/api/v1/roles/*", Action: "PUT"},
			{Name: "角色删除", Resource: "/api/v1/roles/*", Action: "DELETE"},
			{Name: "权限查看", Resource: "/api/v1/permissions", Action: "GET"},
			{Name: "权限创建", Resource: "/api/v1/permissions", Action: "POST"},
		}

		for i := range permissions {
			if err := tx.Where("resource = ? AND action = ?", permissions[i].Resource, permissions[i].Action).
				FirstOrCreate(&permissions[i]).Error; err != nil {
				return fmt.Errorf("seed permission failed: %w", err)
			}
		}

		// 2. 创建超级管理员角色
		adminRole := domain.Role{Name: "超级管理员", Description: "拥有所有权限"}
		if err := tx.Where("name = ?", adminRole.Name).FirstOrCreate(&adminRole).Error; err != nil {
			return fmt.Errorf("seed admin role failed: %w", err)
		}

		// 3. 为超级管理员分配所有权限
		for _, perm := range permissions {
			var count int64
			tx.Table("role_permissions").Where("role_id = ? AND permission_id = ?", adminRole.ID, perm.ID).Count(&count)
			if count == 0 {
				if err := tx.Exec("INSERT INTO role_permissions (role_id, permission_id) VALUES (?, ?)", adminRole.ID, perm.ID).Error; err != nil {
					return fmt.Errorf("assign permission failed: %w", err)
				}
			}
		}

		// 4. 创建默认管理员用户
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
			}
			if err := tx.Create(&adminUser).Error; err != nil {
				return fmt.Errorf("seed admin user failed: %w", err)
			}

			// 5. 为管理员分配超级管理员角色
			if err := tx.Exec("INSERT INTO user_roles (user_id, role_id) VALUES (?, ?)", adminUser.ID, adminRole.ID).Error; err != nil {
				return fmt.Errorf("assign admin role failed: %w", err)
			}
		}

		return nil
	})
}
