package infrastructure

import (
	"context"
	"time"

	"jimu/internal/capabilities/audit/domain"
	dbutil "jimu/internal/kernel/db"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// auditChainHead 审计链头：每个租户一条，写入时锁定该行使链式哈希获得全序，
// 多实例并发写入也能得到一致的链；last_hash 与末尾条目不一致即说明尾部被截断。
type auditChainHead struct {
	TenantID  uint64    `gorm:"primaryKey;column:tenant_id"`
	LastHash  string    `gorm:"column:last_hash;size:64;not null;default:''"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (auditChainHead) TableName() string { return "audit_chain_head" }

// chainLogs 在事务内为待写入条目补齐链式哈希，并推进链头。
func chainLogs(tx *gorm.DB, logs []domain.AuditLog, secret []byte) error {
	if len(logs) == 0 {
		return nil
	}

	grouped := map[uint64][]int{}
	for i := range logs {
		tenantID := logs[i].TenantID
		grouped[tenantID] = append(grouped[tenantID], i)
	}

	for tenantID, indexes := range grouped {
		// 链头行可能尚不存在：ON CONFLICT DO NOTHING 幂等创建后再锁定
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).
			Create(&auditChainHead{TenantID: tenantID}).Error; err != nil {
			return err
		}
		var head auditChainHead
		if err := dbutil.LockRow(tx, &head, tenantID).Error; err != nil {
			return err
		}

		lastHash := head.LastHash
		for _, i := range indexes {
			// 显式设置秒级 UTC 时间，确保入库值与哈希载荷一致（GORM 不会覆盖非零值）
			if logs[i].CreatedAt.IsZero() {
				logs[i].CreatedAt = domain.NormalizeCreatedAt(time.Now())
			}
			logs[i].PrevHash = lastHash
			logs[i].EntryHash = logs[i].ComputeEntryHash(secret)
			lastHash = logs[i].EntryHash
		}

		if err := tx.Model(&auditChainHead{}).
			Where("tenant_id = ?", tenantID).
			Update("last_hash", lastHash).Error; err != nil {
			return err
		}
	}
	return nil
}

// ListForVerify 按 id 升序返回待校验条目
func (r *mysqlAuditRepository) ListForVerify(ctx context.Context, tenantID uint64, fromID, toID uint64, limit int) ([]domain.AuditLog, error) {
	if limit <= 0 {
		limit = 1000
	}
	db := r.db.WithContext(ctx).Model(&domain.AuditLog{}).Order("id ASC").Limit(limit)
	if tenantID != 0 {
		db = db.Where("tenant_id = ?", tenantID)
	}
	if fromID > 0 {
		db = db.Where("id >= ?", fromID)
	}
	if toID > 0 {
		db = db.Where("id <= ?", toID)
	}
	var logs []domain.AuditLog
	if err := db.Find(&logs).Error; err != nil {
		return nil, err
	}
	for i := range logs {
		deserializeChanges(&logs[i])
	}
	return logs, nil
}

// ChainHead 返回该租户审计链最新哈希
func (r *mysqlAuditRepository) ChainHead(ctx context.Context, tenantID uint64) (string, error) {
	var head auditChainHead
	err := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID).First(&head).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", nil
		}
		return "", err
	}
	return head.LastHash, nil
}
