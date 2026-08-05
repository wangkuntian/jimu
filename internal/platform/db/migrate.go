package db

import (
	"database/sql"
	"fmt"
	"time"

	"jimu/internal/config"
	"jimu/internal/platform/logger"

	"github.com/pressly/goose/v3"
	"gorm.io/gorm"
)

// Migrate 执行数据库迁移（兼容旧接口，无重试）
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

	return runMigration(sqlDB, direction)
}

// MigrateWithRetry 带重试的数据库迁移
func MigrateWithRetry(cfg config.DBConfig, log *logger.Logger, direction string) error {
	maxRetries := cfg.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 5
	}
	interval := cfg.RetryIntervalSec
	if interval <= 0 {
		interval = 3
	}

	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		if err := Migrate(cfg, direction); err != nil {
			lastErr = err
			if log != nil {
				log.Warn("retrying database migration",
					"direction", direction,
					"attempt", attempt,
					"max_retries", maxRetries,
					"error", err.Error(),
				)
			}
			time.Sleep(time.Duration(interval) * time.Second)
			continue
		}
		return nil
	}
	return fmt.Errorf("migration %s failed after %d attempts: %w", direction, maxRetries, lastErr)
}

func runMigration(sqlDB *sql.DB, direction string) error {
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
