package db

import (
	"fmt"

	"jimu/internal/config"
	"jimu/internal/platform/logger"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func pgDSN(cfg config.DBConfig) string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable TimeZone=Asia/Shanghai",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database,
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
	return db, nil
}
