package scheduler

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// jobModel 数据库映射模型
type jobModel struct {
	ID        string         `gorm:"primaryKey;size:64" json:"id"`
	Name      string         `gorm:"size:128;not null" json:"name"`
	Cron      string         `gorm:"size:64;not null" json:"cron"`
	Enabled   bool           `gorm:"default:true" json:"enabled"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 表名
func (jobModel) TableName() string { return "scheduled_jobs" }

// MySQLStore 基于 MySQL 的任务定义存储
type MySQLStore struct {
	db *gorm.DB
}

// NewMySQLStore 创建 MySQL 存储
func NewMySQLStore(db *gorm.DB) *MySQLStore {
	return &MySQLStore{db: db}
}

// List 列出所有任务定义
func (s *MySQLStore) List(ctx context.Context) ([]JobDef, error) {
	var models []jobModel
	if err := s.db.WithContext(ctx).Find(&models).Error; err != nil {
		return nil, err
	}
	result := make([]JobDef, 0, len(models))
	for _, m := range models {
		result = append(result, JobDef{
			ID:        m.ID,
			Name:      m.Name,
			Cron:      m.Cron,
			Enabled:   m.Enabled,
			CreatedAt: m.CreatedAt,
			UpdatedAt: m.UpdatedAt,
		})
	}
	return result, nil
}

// Save 保存任务定义（新增或更新）
func (s *MySQLStore) Save(ctx context.Context, job JobDef) error {
	return s.db.WithContext(ctx).Save(&jobModel{
		ID:      job.ID,
		Name:    job.Name,
		Cron:    job.Cron,
		Enabled: job.Enabled,
	}).Error
}

// Delete 删除任务定义
func (s *MySQLStore) Delete(ctx context.Context, id string) error {
	return s.db.WithContext(ctx).Where("id = ?", id).Delete(&jobModel{}).Error
}

var _ Store = (*MySQLStore)(nil)
