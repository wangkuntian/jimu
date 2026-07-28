package testutil

import (
	"fmt"

	"jimu/internal/config"

	"github.com/pressly/goose/v3"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// TestDB 测试数据库连接
type TestDB struct {
	*gorm.DB
}

// NewTestDB 创建测试数据库连接
func NewTestDB(cfg config.DBConfig) (*TestDB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect test database: %w", err)
	}

	return &TestDB{DB: db}, nil
}

// Migrate 执行迁移
func (tdb *TestDB) Migrate() error {
	sqlDB, err := tdb.DB.DB()
	if err != nil {
		return err
	}

	if err := goose.SetDialect("mysql"); err != nil {
		return err
	}

	return goose.Up(sqlDB, "migrations")
}

// Reset 清空所有表数据（保留表结构）
func (tdb *TestDB) Reset(models ...interface{}) error {
	for _, model := range models {
		if err := tdb.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(model).Error; err != nil {
			return fmt.Errorf("failed to reset table: %w", err)
		}
	}
	return nil
}

// Truncate 截断表（更快）
func (tdb *TestDB) Truncate(tableName string) error {
	return tdb.Exec(fmt.Sprintf("TRUNCATE TABLE %s", tableName)).Error
}

// Close 关闭连接
func (tdb *TestDB) Close() error {
	sqlDB, err := tdb.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
