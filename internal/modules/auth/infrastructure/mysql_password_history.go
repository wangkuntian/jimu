package infrastructure

import (
	"context"

	"jimu/internal/modules/auth/domain"

	"gorm.io/gorm"
)

type mysqlPasswordHistoryRepository struct {
	db *gorm.DB
}

// NewMysqlPasswordHistoryRepository 创建密码历史仓储
func NewMysqlPasswordHistoryRepository(db *gorm.DB) domain.PasswordHistoryRepository {
	return &mysqlPasswordHistoryRepository{db: db}
}

func (r *mysqlPasswordHistoryRepository) Add(ctx context.Context, tenantID, userID uint64, passwordHash string) error {
	return r.db.WithContext(ctx).Create(&domain.PasswordHistory{
		TenantID:     tenantID,
		UserID:       userID,
		PasswordHash: passwordHash,
	}).Error
}

func (r *mysqlPasswordHistoryRepository) ListRecentHashes(ctx context.Context, userID uint64, limit int) ([]string, error) {
	if limit <= 0 {
		return nil, nil
	}
	var hashes []string
	err := r.db.WithContext(ctx).Model(&domain.PasswordHistory{}).
		Where("user_id = ?", userID).
		Order("id DESC").
		Limit(limit).
		Pluck("password_hash", &hashes).Error
	return hashes, err
}

// Trim 仅保留最近 keep 条：先取第 keep 新的记录 ID，再删除更早的记录。
// 刻意不用 `id NOT IN (SELECT ... LIMIT n)`——MySQL/MariaDB 不支持该写法（错误 1235）。
func (r *mysqlPasswordHistoryRepository) Trim(ctx context.Context, userID uint64, keep int) error {
	if keep <= 0 {
		return nil
	}
	var cutoff uint64
	if err := r.db.WithContext(ctx).Model(&domain.PasswordHistory{}).
		Where("user_id = ?", userID).
		Order("id DESC").
		Offset(keep-1).
		Limit(1).
		Pluck("id", &cutoff).Error; err != nil {
		return err
	}
	if cutoff == 0 {
		return nil // 历史条数不足 keep，无需清理
	}
	return r.db.WithContext(ctx).
		Where("user_id = ? AND id < ?", userID, cutoff).
		Delete(&domain.PasswordHistory{}).Error
}
