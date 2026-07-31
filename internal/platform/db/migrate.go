package db

import (
	"database/sql"
	"fmt"

	"jimu/internal/config"

	"github.com/pressly/goose/v3"
	"gorm.io/gorm"
)

// Migrate 执行数据库迁移
func Migrate(cfg config.DBConfig, direction string) error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database)

	sqlDB, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer func() { _ = sqlDB.Close() }()

	if err := goose.SetDialect("mysql"); err != nil {
		return fmt.Errorf("failed to set dialect: %w", err)
	}

	switch direction {
	case "up":
		return goose.Up(sqlDB, "migrations")
	case "up-by-one":
		return goose.UpByOne(sqlDB, "migrations")
	case "down":
		return goose.Down(sqlDB, "migrations")
	case "redo":
		return goose.Redo(sqlDB, "migrations")
	case "status":
		return goose.Status(sqlDB, "migrations")
	case "reset":
		return goose.Reset(sqlDB, "migrations")
	default:
		return fmt.Errorf("unknown direction: %s", direction)
	}
}

// AutoMigrate 使用 Gorm 自动迁移（开发用）
func AutoMigrate(db *gorm.DB, models ...interface{}) error {
	return db.AutoMigrate(models...)
}
