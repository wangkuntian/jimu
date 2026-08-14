package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"jimu/internal/config"
	"jimu/internal/platform/logger"

	_ "github.com/jackc/pgx/v5/stdlib" // 注册 pgx driver，供 goose postgres 迁移使用
	"github.com/pressly/goose/v3"
	"gorm.io/gorm"
)

// Migrate 执行数据库迁移（兼容旧接口，无重试）
func Migrate(cfg config.DBConfig, direction string) error {
	driver, dsnStr, err := sqlDriverAndDSN(cfg)
	if err != nil {
		return err
	}

	sqlDB, err := sql.Open(driver, dsnStr)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer func() { _ = sqlDB.Close() }()

	return runMigration(sqlDB, cfg, direction)
}

// sqlDriverAndDSN 根据 Driver 配置返回 database/sql driver 名与 DSN
func sqlDriverAndDSN(cfg config.DBConfig) (string, string, error) {
	dialect := strings.ToLower(cfg.Driver)
	switch dialect {
	case "postgres", "postgresql":
		return "pgx", pgDSN(cfg), nil
	case "", "mysql", "mariadb":
		return "mysql", mysqlDSN(cfg), nil
	default:
		return "", "", fmt.Errorf("unsupported db driver: %s", cfg.Driver)
	}
}

func mysqlDSN(cfg config.DBConfig) string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database)
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

// MigrationDir 定位 MySQL 迁移目录：从本文件源码路径向上找项目根的 migrations，
// 不依赖工作目录，go test 在包目录运行也能找到。
func MigrationDir() string {
	return migrationDir("")
}

// PostgresMigrationDir 定位 PostgreSQL 迁移目录（migrations/postgres）
func PostgresMigrationDir() string {
	return migrationDir("postgres")
}

// migrationDir 按子目录定位迁移目录
func migrationDir(sub string) string {
	if _, file, _, ok := runtime.Caller(0); ok {
		if dir := findUp(filepath.Dir(file), "migrations"); dir != "" {
			if sub == "" {
				return dir
			}
			return filepath.Join(dir, sub)
		}
	}
	if sub == "" {
		return "migrations"
	}
	return filepath.Join("migrations", sub)
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

func runMigration(sqlDB *sql.DB, cfg config.DBConfig, direction string) error {
	dialect := cfg.Dialect()
	if err := goose.SetDialect(dialect); err != nil {
		return fmt.Errorf("failed to set dialect: %w", err)
	}

	var dir string
	if dialect == "postgres" {
		dir = PostgresMigrationDir()
	} else {
		dir = MigrationDir()
	}

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
