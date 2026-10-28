package app

import (
	stderrors "errors"
	"fmt"

	"jimu/internal/capabilities/access/domain"
	tenantDomain "jimu/internal/capabilities/tenant/domain"
	userdomain "jimu/internal/capabilities/user/domain"
	"jimu/internal/contract"
	"jimu/internal/kernel/access"
	"jimu/internal/kernel/tenant"

	"github.com/casbin/casbin/v3"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// RunSeed 插入初始数据
// 管理员密码优先从 ADMIN_PASSWORD_FILE 读取，回退到 ADMIN_PASSWORD 环境变量
// 权限点来自启用集各能力的 Descriptor.Permissions（能力自声明，未启用不种）
func RunSeed(db *gorm.DB, caps []contract.Descriptor) error {
	adminPassword := AdminPassword()
	if adminPassword == "" {
		return fmt.Errorf("ADMIN_PASSWORD or ADMIN_PASSWORD_FILE is required")
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

		permissionIDs := make(map[contract.ProvisioningPermission]uint64, len(permissions))
		for _, perm := range permissions {
			permissionIDs[contract.ProvisioningPermission{Resource: perm.Resource, Action: perm.Action}] = perm.ID
		}

		// 3. 幂等创建平台系统角色。
		platformRoles := make(map[string]*domain.Role, len(contract.PlatformSystemRoles()))
		for _, spec := range contract.PlatformSystemRoles() {
			role, err := ensureSystemRole(tx, tenant.DefaultTenantID, spec)
			if err != nil {
				return fmt.Errorf("seed system role %s failed: %w", spec.Code, err)
			}
			platformRoles[spec.Code] = role

			// 4. 超管授予当前全部权限；其他系统角色只看显式权限集合。
			ids := make([]uint64, 0, len(permissions))
			if spec.AllPermissions {
				for _, perm := range permissions {
					ids = append(ids, perm.ID)
				}
			} else {
				for _, permission := range spec.Permissions {
					if id := permissionIDs[permission]; id != 0 {
						ids = append(ids, id)
					}
				}
			}
			if err := assignRolePermissions(tx, role.ID, ids); err != nil {
				return fmt.Errorf("assign permissions for %s failed: %w", spec.Code, err)
			}
		}

		// 5. 创建默认管理员用户
		var adminUser userdomain.User
		result := tx.Unscoped().Where("username = ?", "admin").First(&adminUser)
		if stderrors.Is(result.Error, gorm.ErrRecordNotFound) {
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
		} else if result.Error != nil {
			return fmt.Errorf("find admin user failed: %w", result.Error)
		}

		// 6. 即使 admin 已存在，也确保超管关联存在；不恢复其 status/deleted_at。
		if err := ensureAdminRole(tx, adminUser.ID, platformRoles[contract.RoleCodePlatformSuperAdmin].ID); err != nil {
			return fmt.Errorf("assign admin role failed: %w", err)
		}
		active, err := hasActivePlatformSuperAdmin(tx)
		if err != nil {
			return fmt.Errorf("check active platform superadmin failed: %w", err)
		}
		if !active {
			return fmt.Errorf("no active platform superadmin after seed")
		}

		return nil
	})
}

func ensureSystemRole(tx *gorm.DB, tenantID uint64, spec contract.SystemRoleSpec) (*domain.Role, error) {
	var role domain.Role
	err := tx.Where("tenant_id = ? AND code = ?", tenantID, spec.Code).First(&role).Error
	if err != nil && !stderrors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		role = domain.Role{
			TenantID: tenantID,
			Name:     spec.Name,
			Code:     spec.Code,
			Scope:    spec.Scope,
			IsSystem: true,
			Status:   1,
		}
	}
	role.Name = spec.Name
	role.Description = spec.Description
	role.Code = spec.Code
	role.Scope = spec.Scope
	role.IsSystem = true
	role.Status = 1
	if role.ID == 0 {
		if err := tx.Create(&role).Error; err != nil {
			return nil, err
		}
	} else if err := tx.Save(&role).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

func assignRolePermissions(tx *gorm.DB, roleID uint64, permissionIDs []uint64) error {
	if err := tx.Exec("DELETE FROM role_permissions WHERE role_id = ?", roleID).Error; err != nil {
		return fmt.Errorf("clear role permissions failed: %w", err)
	}
	for _, permissionID := range permissionIDs {
		if err := tx.Exec("INSERT INTO role_permissions (role_id, permission_id) VALUES (?, ?)", roleID, permissionID).Error; err != nil {
			return fmt.Errorf("assign permission failed: %w", err)
		}
	}
	return nil
}

func ensureAdminRole(tx *gorm.DB, userID, roleID uint64) error {
	var count int64
	if err := tx.Table("user_roles").Where("user_id = ? AND role_id = ?", userID, roleID).Count(&count).Error; err != nil {
		return fmt.Errorf("check admin role failed: %w", err)
	}
	if count == 0 {
		if err := tx.Exec("INSERT INTO user_roles (user_id, role_id) VALUES (?, ?)", userID, roleID).Error; err != nil {
			return err
		}
	}
	return nil
}

func hasActivePlatformSuperAdmin(tx *gorm.DB) (bool, error) {
	var count int64
	err := tx.Table("users").
		Joins("JOIN user_roles ur ON ur.user_id = users.id").
		Joins("JOIN roles ON roles.id = ur.role_id").
		Where("users.status = 1 AND users.deleted_at IS NULL").
		Where("roles.status = 1 AND roles.deleted_at IS NULL AND roles.code = ?", contract.RoleCodePlatformSuperAdmin).
		Distinct("users.id").
		Count(&count).Error
	return count > 0, err
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
		_, err := enforcer.AddPolicy(access.RoleSubject(p.RoleID), p.Resource, p.Action)
		if err != nil {
			return fmt.Errorf("add policy (%d, %s, %s): %w", p.RoleID, p.Resource, p.Action, err)
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

// NoAccountLogger 启动自检所需的最小日志面（*logger.Logger 满足）。
type NoAccountLogger interface {
	Errorw(msg string, keysAndValues ...interface{})
}

// WarnIfNoAccounts 是结构性种子被跳过时的最后一道自检：
// 库中一个账号都没有意味着 API 没有任何可用登录（生产关闭公开注册时即为死局），
// 此时给出 error 级提示与下一步；返回是否已提示。
//
// 表不存在或查询失败一律静默降级：容器启动早于 CLI 迁移是已知且允许的时序，
// 服务启动不得因为缺 ADMIN_PASSWORD 或表未建而失败。
// 探针走 Discard 会话并先做建表预检：生产 GORM logger 会把每条报错语句打成 error
// （kernel/db/gorm_logger.go 的 Trace），若直接查询缺失的 users 表，容器每次启动都会
// 多出一行 "db query failed" —— 那是把允许的时序渲染成故障。
func WarnIfNoAccounts(db *gorm.DB, log NoAccountLogger) bool {
	if db == nil || log == nil {
		return false
	}
	probe := db.Session(&gorm.Session{Logger: gormlogger.Discard})
	if !probe.Migrator().HasTable(&userdomain.User{}) {
		return false
	}
	var count int64
	if err := probe.Model(&userdomain.User{}).Count(&count).Error; err != nil {
		return false
	}
	if count > 0 {
		return false
	}
	log.Errorw("no account exists and structural seed is disabled; the API has no usable login",
		"hint", "set ADMIN_PASSWORD or ADMIN_PASSWORD_FILE and restart the server, or enable auth.public_registration")
	return true
}

// 确保接口实现
var _ = func() *casbin.Enforcer { return nil }
