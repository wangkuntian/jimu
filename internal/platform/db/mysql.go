package db

import (
	"context"
	"fmt"
	"time"

	"jimu/internal/config"
	"jimu/internal/platform/logger"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"gorm.io/plugin/dbresolver"
)

// New 创建数据库连接（带重试和连接池配置）
func New(cfg config.DBConfig, log *logger.Logger) (*gorm.DB, error) {
	db, err := ConnectWithRetry(cfg, log)
	if err != nil {
		return nil, err
	}
	return db, nil
}

// ConnectWithRetry 带重试的数据库连接（使用自定义 logger 支持慢查询告警）
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
		db, err = open(cfg, log)
		if err == nil {
			if pingErr := pingDB(context.Background(), db); pingErr == nil {
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

func dsn(cfg config.DBConfig, host string, port int) string {
	if host == "" {
		host = cfg.Host
	}
	if port == 0 {
		port = cfg.Port
	}
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.User, cfg.Password, host, port, cfg.Database)
}

func open(cfg config.DBConfig, log ...*logger.Logger) (*gorm.DB, error) {
	gormCfg := &gorm.Config{}
	if len(log) > 0 && log[0] != nil {
		gormCfg.Logger = NewGormLogger(log[0], SlowQueryThreshold)
	} else {
		gormCfg.Logger = gormlogger.Default.LogMode(gormlogger.Silent)
	}
	db, err := gorm.Open(mysql.Open(dsn(cfg, "", 0)), gormCfg)
	if err != nil {
		return nil, err
	}
	// 雪花 ID 主键注入（InitSnowflake 未调用时 no-op，回退数据库自增）
	RegisterSnowflakeHook(db)

	// 配置读写分离（如果有从库）
	if len(cfg.ReadHosts) > 0 {
		sources := []gorm.Dialector{mysql.Open(dsn(cfg, "", 0))}
		var replicas []gorm.Dialector
		for i, host := range cfg.ReadHosts {
			port := 3306
			if i < len(cfg.ReadPorts) {
				port = cfg.ReadPorts[i]
			}
			replicas = append(replicas, mysql.Open(dsn(cfg, host, port)))
		}

		resolverCfg := dbresolver.Config{
			Sources:  sources,
			Replicas: replicas,
			Policy:   dbresolver.RandomPolicy{},
		}

		if err := db.Use(dbresolver.Register(resolverCfg).
			SetConnMaxIdleTime(time.Duration(cfg.ConnMaxIdleTimeSec) * time.Second).
			SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetimeSec) * time.Second).
			SetMaxIdleConns(cfg.MaxIdle).
			SetMaxOpenConns(cfg.MaxOpen),
		); err != nil {
			return nil, fmt.Errorf("register dbresolver: %w", err)
		}
	}

	return db, nil
}

func pingDB(ctx context.Context, db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
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
