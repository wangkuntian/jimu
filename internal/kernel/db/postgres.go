package db

import (
	"fmt"
	"time"

	"jimu/internal/config"
	"jimu/internal/kernel/logger"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"gorm.io/plugin/dbresolver"
)

func pgDSN(cfg config.DBConfig) string {
	return pgDSNHost(cfg, "", 0)
}

// pgDSNHost 生成 DSN，host/port 为空或 0 时回退到主库配置
func pgDSNHost(cfg config.DBConfig, host string, port int) string {
	if host == "" {
		host = cfg.Host
	}
	if port == 0 {
		port = cfg.Port
	}
	// 时间统一按 UTC 存储与读取（与 MySQL 侧约定一致）
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable TimeZone=UTC",
		host, port, cfg.User, cfg.Password, cfg.Database,
	)
}

func openPostgres(cfg config.DBConfig, log *logger.Logger) (*gorm.DB, error) {
	gormCfg := &gorm.Config{}
	if log != nil {
		gormCfg.Logger = NewGormLogger(log, SlowQueryThreshold)
	} else {
		gormCfg.Logger = gormlogger.Default.LogMode(gormlogger.Silent)
	}
	db, err := gorm.Open(postgres.Open(pgDSN(cfg)), gormCfg)
	if err != nil {
		return nil, err
	}
	RegisterSnowflakeHook(db)

	// 配置读写分离（如果有从库），与 MySQL 行为一致
	if len(cfg.ReadHosts) > 0 {
		sources := []gorm.Dialector{postgres.Open(pgDSN(cfg))}
		var replicas []gorm.Dialector
		for i, host := range cfg.ReadHosts {
			port := 5432
			if i < len(cfg.ReadPorts) {
				port = cfg.ReadPorts[i]
			}
			replicas = append(replicas, postgres.Open(pgDSNHost(cfg, host, port)))
		}

		if err := db.Use(dbresolver.Register(dbresolver.Config{
			Sources:  sources,
			Replicas: replicas,
			Policy:   dbresolver.RandomPolicy{},
		}).
			SetConnMaxIdleTime(time.Duration(cfg.ConnMaxIdleTimeSec) * time.Second).
			SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetimeSec) * time.Second).
			SetMaxIdleConns(cfg.MaxIdle).
			SetMaxOpenConns(cfg.MaxOpen),
		); err != nil {
			return nil, fmt.Errorf("register dbresolver: %w", err)
		}
	}

	if err := attachBreaker(db, cfg.Breaker); err != nil {
		return nil, err
	}

	return db, nil
}
