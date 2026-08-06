package db

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// CleanupConfig 数据清理配置
type CleanupConfig struct {
	RetentionDays int           // 保留天数，超过此天数的软删除数据将被清理
	BatchSize     int           // 每批清理数量
	Tables        []CleanupTable // 要清理的表
}

// CleanupTable 要清理的表配置
type CleanupTable struct {
	Model       interface{} // GORM 模型
	DeletedAt   string      // 软删除字段名，默认 "deleted_at"
}

// DefaultCleanupConfig 返回默认清理配置
func DefaultCleanupConfig() CleanupConfig {
	return CleanupConfig{
		RetentionDays: 90,
		BatchSize:     100,
	}
}

// CleanupResult 清理结果
type CleanupResult struct {
	Table   string `json:"table"`
	Deleted int64  `json:"deleted"`
}

// CleanupService 数据清理服务
type CleanupService struct {
	db     *gorm.DB
	config CleanupConfig
}

// NewCleanupService 创建清理服务
func NewCleanupService(db *gorm.DB, config CleanupConfig) *CleanupService {
	if config.RetentionDays <= 0 {
		config.RetentionDays = 90
	}
	if config.BatchSize <= 0 {
		config.BatchSize = 100
	}
	return &CleanupService{db: db, config: config}
}

// Run 执行清理（硬删除超过保留期的软删除数据）
func (s *CleanupService) Run(ctx context.Context) ([]CleanupResult, error) {
	var results []CleanupResult
	cutoff := time.Now().AddDate(0, 0, -s.config.RetentionDays)

	for _, table := range s.config.Tables {
		deleted, err := s.cleanTable(ctx, table, cutoff)
		if err != nil {
			return results, err
		}
		results = append(results, CleanupResult{
			Table:   tableName(table.Model),
			Deleted: deleted,
		})
	}
	return results, nil
}

func (s *CleanupService) cleanTable(ctx context.Context, table CleanupTable, cutoff time.Time) (int64, error) {
	deletedAtCol := table.DeletedAt
	if deletedAtCol == "" {
		deletedAtCol = "deleted_at"
	}

	var totalDeleted int64
	for {
		result := s.db.WithContext(ctx).
			Where(deletedAtCol+" IS NOT NULL AND "+deletedAtCol+" < ?", cutoff).
			Limit(s.config.BatchSize).
			Delete(table.Model)
		if result.Error != nil {
			return totalDeleted, result.Error
		}
		if result.RowsAffected == 0 {
			break
		}
		totalDeleted += result.RowsAffected
	}
	return totalDeleted, nil
}

func tableName(model interface{}) string {
	// 简化实现：返回类型名
	if t, ok := model.(interface{ TableName() string }); ok {
		return t.TableName()
	}
	return "unknown"
}
