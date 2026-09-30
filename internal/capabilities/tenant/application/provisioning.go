package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	stderrors "errors"
	"fmt"
	"strings"

	tenantdomain "jimu/internal/capabilities/tenant/domain"
	"jimu/internal/contract"
	dbctx "jimu/internal/kernel/db"
	"jimu/internal/kernel/tenant"
	"jimu/internal/shared/errors"

	mysqlerr "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

// ProvisioningConfig 开通式注册配置（tenant 自有的输入视图）。
//
// `auth.provisioning` 段由 auth 能力拥有（设计 §8 ¶2 不拆段），但 tenant 被 auth 依赖，
// **不得** import auth 能力类型，故在此定义同形类型，由组合根从 auth 段构造后传入。
type ProvisioningConfig struct {
	Enabled   bool
	OwnerRole string
	Roles     []ProvisionRoleTemplate
}

// ProvisionRoleTemplate 开通租户时初始化的角色模板（见 ProvisioningConfig）。
type ProvisionRoleTemplate struct {
	Name        string
	Description string
	Permissions []ProvisionPermission
}

// ProvisionPermission 模板角色绑定的全局权限。
type ProvisionPermission struct {
	Resource string
	Action   string
}

// GormTenantProvisioner 基于单事务的租户开通实现（实现 contract.TenantProvisioner）。
type GormTenantProvisioner struct {
	db    *gorm.DB
	cfg   ProvisioningConfig
	users contract.AccountRepository
	roles contract.ProvisioningRoleStore
}

func NewGormTenantProvisioner(db *gorm.DB, cfg ProvisioningConfig, users contract.AccountRepository, roles contract.ProvisioningRoleStore) *GormTenantProvisioner {
	return &GormTenantProvisioner{db: db, cfg: cfg, users: users, roles: roles}
}

// Provision 创建租户 + owner 用户 + 模板角色（单事务，任一步失败整体回滚）。
// 模板权限引用全局权限表，缺失的条目跳过（不阻塞开通）。租户编码统一转小写存储。
func (p *GormTenantProvisioner) Provision(ctx context.Context, params contract.ProvisionRequest) (*contract.ProvisionResult, error) {
	if params.TenantName == "" {
		return nil, errors.New(errors.CodeInvalidParam, "tenant_name is required")
	}
	tenantCode := tenant.NormalizeCode(params.TenantCode)
	if tenantCode != "" && !tenant.ValidCode(tenantCode) {
		return nil, errors.New(errors.CodeTenantCodeFormat, "tenant code must match [a-zA-Z0-9_-] (1-64 chars)")
	}

	var result *contract.ProvisionResult
	err := p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txCtx := dbctx.WithTransaction(ctx, tx)
		// 1. 创建租户（编码唯一；自动生成时在事务内重试冲突）
		tenant := &tenantdomain.Tenant{Name: params.TenantName, Status: 1}
		if tenantCode != "" {
			var existing tenantdomain.Tenant
			if err := tx.Where("code = ?", tenantCode).First(&existing).Error; err == nil {
				return errors.New(errors.CodeTenantExists, "tenant code already exists")
			} else if !isRecordNotFound(err) {
				return err
			}
			tenant.Code = tenantCode
		} else {
			code, err := generateTenantCode(tx)
			if err != nil {
				return err
			}
			tenant.Code = code
		}
		if err := tx.Create(tenant).Error; err != nil {
			return err
		}

		// 2. 创建 owner 用户（归属新租户）
		user := &contract.Account{
			Username: params.Username,
			Password: params.PasswordHash,
			Email:    params.Email,
			Phone:    params.Phone,
			Status:   1,
			TenantID: tenant.ID,
		}
		if err := p.users.Create(txCtx, user); err != nil {
			if isDuplicateKeyErr(err) {
				return errors.Wrap(errors.CodeUserExists, "username already exists", err)
			}
			return err
		}

		// 3. 按模板初始化角色 + 权限绑定；owner 绑定 owner_role（缺省第一个角色）
		rolesCfg := contract.AuthProvisioningConfig{Enabled: p.cfg.Enabled, OwnerRole: p.cfg.OwnerRole}
		for _, role := range p.cfg.Roles {
			r := contract.AuthProvisionRole{Name: role.Name, Description: role.Description}
			for _, perm := range role.Permissions {
				r.Permissions = append(r.Permissions, contract.AuthProvisionPermission{Resource: perm.Resource, Action: perm.Action})
			}
			rolesCfg.Roles = append(rolesCfg.Roles, r)
		}
		ownerRoleID, err := p.roles.ProvisionRoles(txCtx, tenant.ID, rolesCfg)
		if err != nil {
			return err
		}
		if ownerRoleID != 0 {
			if err := p.roles.AssignRole(txCtx, user.ID, ownerRoleID); err != nil {
				return err
			}
		}

		result = &contract.ProvisionResult{
			Tenant: contract.ProvisionedTenant{
				ID:        tenant.ID,
				Code:      tenant.Code,
				Name:      tenant.Name,
				Status:    tenant.Status,
				PlanID:    tenant.PlanID,
				CreatedAt: tenant.CreatedAt,
				UpdatedAt: tenant.UpdatedAt,
			},
			User: contract.ProvisionedUser{
				ID:        user.ID,
				Username:  user.Username,
				Email:     user.Email,
				Phone:     user.Phone,
				Status:    user.Status,
				TenantID:  user.TenantID,
				CreatedAt: user.CreatedAt,
				UpdatedAt: user.UpdatedAt,
			},
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// generateTenantCode 自动生成租户编码（t + 12 位随机 hex），冲突重试最多 5 次
func generateTenantCode(tx *gorm.DB) (string, error) {
	for range 5 {
		b := make([]byte, 6)
		if _, err := rand.Read(b); err != nil {
			return "", fmt.Errorf("generate tenant code: %w", err)
		}
		code := "t" + hex.EncodeToString(b)
		var existing tenantdomain.Tenant
		if err := tx.Where("code = ?", code).First(&existing).Error; isRecordNotFound(err) {
			return code, nil
		} else if err != nil {
			return "", err
		}
	}
	return "", errors.New(errors.CodeInternalError, "failed to generate unique tenant code")
}

func isRecordNotFound(err error) bool {
	return err != nil && (err == gorm.ErrRecordNotFound || err.Error() == gorm.ErrRecordNotFound.Error())
}

func isDuplicateKeyErr(err error) bool {
	if err == nil {
		return false
	}
	if stderrors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	var mysqlErr *mysqlerr.MySQLError
	if stderrors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
		return true
	}
	// sqlite 驱动不翻译错误类型，按文本识别唯一约束冲突
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique constraint failed") || strings.Contains(msg, "duplicate entry")
}
