package db

import (
	"fmt"
	"time"

	"jimu/internal/config"
	"jimu/internal/platform/logger"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// New 创建数据库连接（带重试和连接池配置）
func New(cfg config.DBConfig, log *logger.Logger) (*gorm.DB, error) {
	db, err := ConnectWithRetry(cfg, log)
	if err != nil {
		return nil, err
	}
	return db, nil
}

// ConnectWithRetry 带重试的数据库连接
func ConnectWithRetry(cfg config.DBConfig, log *logger.Logger) (*gorm.DB, error) {
	maxRetries := cfg.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 5
	}
	interval := cfg.RetryIntervalSec
	if interval <= 0 {
		interval = 3
	}

	var db *gorm.DB
	var err error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		db, err = open(cfg)
		if err == nil {
			if pingErr := pingDB(db); pingErr == nil {
				if log != nil {
					log.Info("database connected", "attempt", attempt)
				}
				configurePool(db, cfg)
				return db, nil
			} else {
				err = pingErr
			}
		}

		if log != nil {
			log.Warn("retrying database connection",
				"attempt", attempt,
				"max_retries", maxRetries,
				"interval_sec", interval,
				"error", err.Error(),
			)
		}
		time.Sleep(time.Duration(interval) * time.Second)
	}

	return nil, fmt.Errorf("database connection failed after %d attempts: %w", maxRetries, err)
}

func open(cfg config.DBConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database)

	return gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Info),
	})
}

func pingDB(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}

func configurePool(db *gorm.DB, cfg config.DBConfig) {
	sqlDB, err := db.DB()
	if err != nil {
		return
	}
	sqlDB.SetMaxOpenConns(cfg.MaxOpen)
	sqlDB.SetMaxIdleConns(cfg.MaxIdle)
	if cfg.ConnMaxLifetimeSec > 0 {
		sqlDB.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetimeSec) * time.Second)
	}
	if cfg.ConnMaxIdleTimeSec > 0 {
		sqlDB.SetConnMaxIdleTime(time.Duration(cfg.ConnMaxIdleTimeSec) * time.Second)
	}
}
