package access

import (
	"context"
	"errors"

	"jimu/internal/capabilities/access/domain"
	"jimu/internal/contract"
	dbctx "jimu/internal/kernel/db"

	"gorm.io/gorm"
)

const ProvisioningRolePortName = "access.provisioning"

type provisioningRoleStore struct{ db *gorm.DB }

func NewProvisioningRoleStore(db *gorm.DB) contract.ProvisioningRoleStore {
	return provisioningRoleStore{db: db}
}

func (s provisioningRoleStore) transaction(ctx context.Context) *gorm.DB {
	if tx, ok := dbctx.TransactionFromContext(ctx); ok {
		return tx.WithContext(ctx)
	}
	return s.db.WithContext(ctx)
}

func (s provisioningRoleStore) ProvisionRoles(ctx context.Context, tenantID uint64, cfg contract.AuthProvisioningConfig) (uint64, error) {
	tx := s.transaction(ctx)
	var ownerRoleID uint64
	for _, template := range cfg.Roles {
		role := domain.Role{Name: template.Name, Description: template.Description, Status: 1, TenantID: tenantID}
		if err := tx.Create(&role).Error; err != nil {
			return 0, err
		}
		for _, perm := range template.Permissions {
			var gp domain.Permission
			if err := tx.Where("resource = ? AND action = ?", perm.Resource, perm.Action).First(&gp).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					continue
				}
				return 0, err
			}
			if err := tx.Exec("INSERT INTO role_permissions (role_id, permission_id) VALUES (?, ?)", role.ID, gp.ID).Error; err != nil {
				return 0, err
			}
		}
		if template.Name == cfg.OwnerRole {
			ownerRoleID = role.ID
		} else if ownerRoleID == 0 {
			ownerRoleID = role.ID
		}
	}
	return ownerRoleID, nil
}

func (s provisioningRoleStore) AssignRole(ctx context.Context, userID, roleID uint64) error {
	return s.transaction(ctx).Exec("INSERT INTO user_roles (user_id, role_id) VALUES (?, ?)", userID, roleID).Error
}

var _ contract.ProvisioningRoleStore = provisioningRoleStore{}
