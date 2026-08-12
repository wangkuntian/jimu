package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
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

// MigrationDir 定位迁移目录：从本文件源码路径向上找项目根的 migrations，
// 不依赖工作目录，go test 在包目录运行也能找到。
func MigrationDir() string {
	if _, file, _, ok := runtime.Caller(0); ok {
		if dir := findUp(filepath.Dir(file), "migrations"); dir != "" {
			return dir
		}
	}
	return "migrations"
}

// findUp 从 start 逐级向父目录查找包含 target 的目录
func findUp(start, target string) string {
	dir := start
	for {
		if isDir(filepath.Join(dir, target)) {
			return filepath.Join(dir, target)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func runMigration(sqlDB *sql.DB, direction string) error {
	if err := goose.SetDialect("mysql"); err != nil {
		return fmt.Errorf("failed to set dialect: %w", err)
	}

	dir := MigrationDir()
	switch direction {
	case "up":
		return goose.Up(sqlDB, dir)
	case "up-by-one":
		return goose.UpByOne(sqlDB, dir)
	case "down":
		return goose.Down(sqlDB, dir)
	case "redo":
		return goose.Redo(sqlDB, dir)
	case "status":
		return goose.Status(sqlDB, dir)
	case "reset":
		return goose.Reset(sqlDB, dir)
	default:
		return fmt.Errorf("unknown direction: %s", direction)
	}
}

// AutoMigrate 使用 Gorm 自动迁移（开发用）
func AutoMigrate(db *gorm.DB, models ...interface{}) error {
	return db.AutoMigrate(models...)
}
