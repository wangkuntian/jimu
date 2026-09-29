package infrastructure

import (
	"context"
	"time"

	"jimu/internal/capabilities/passkey/domain"

	"gorm.io/gorm"
)

type mysqlWebAuthnCredentialRepository struct {
	db *gorm.DB
}

// NewMysqlWebAuthnCredentialRepository 创建 WebAuthn 凭证仓储
func NewMysqlWebAuthnCredentialRepository(db *gorm.DB) domain.WebAuthnCredentialRepository {
	return &mysqlWebAuthnCredentialRepository{db: db}
}

func (r *mysqlWebAuthnCredentialRepository) Create(ctx context.Context, credential *domain.WebAuthnCredential) error {
	return r.db.WithContext(ctx).Create(credential).Error
}

func (r *mysqlWebAuthnCredentialRepository) FindByCredentialID(ctx context.Context, credentialID string) (*domain.WebAuthnCredential, error) {
	var credential domain.WebAuthnCredential
	if err := r.db.WithContext(ctx).Where("credential_id = ?", credentialID).First(&credential).Error; err != nil {
		return nil, err
	}
	return &credential, nil
}

func (r *mysqlWebAuthnCredentialRepository) ListByUser(ctx context.Context, tenantID, userID uint64) ([]domain.WebAuthnCredential, error) {
	var credentials []domain.WebAuthnCredential
	db := r.db.WithContext(ctx).Model(&domain.WebAuthnCredential{}).Where("user_id = ?", userID)
	if tenantID != 0 {
		db = db.Where("tenant_id = ?", tenantID)
	}
	err := db.Order("id DESC").Find(&credentials).Error
	return credentials, err
}

func (r *mysqlWebAuthnCredentialRepository) CountByUser(ctx context.Context, userID uint64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&domain.WebAuthnCredential{}).Where("user_id = ?", userID).Count(&count).Error
	return count, err
}

func (r *mysqlWebAuthnCredentialRepository) Touch(ctx context.Context, id uint64, signCount uint32, backupState bool, usedAt time.Time) error {
	return r.db.WithContext(ctx).Model(&domain.WebAuthnCredential{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"sign_count":   signCount,
			"backup_state": backupState,
			"last_used_at": usedAt,
		}).Error
}

func (r *mysqlWebAuthnCredentialRepository) UpdateName(ctx context.Context, tenantID, userID, id uint64, name string) error {
	db := r.db.WithContext(ctx).Model(&domain.WebAuthnCredential{}).Where("id = ? AND user_id = ?", id, userID)
	if tenantID != 0 {
		db = db.Where("tenant_id = ?", tenantID)
	}
	return db.Update("name", name).Error
}

func (r *mysqlWebAuthnCredentialRepository) Delete(ctx context.Context, tenantID, userID, id uint64) error {
	db := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID)
	if tenantID != 0 {
		db = db.Where("tenant_id = ?", tenantID)
	}
	return db.Delete(&domain.WebAuthnCredential{}).Error
}
