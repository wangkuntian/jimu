package application

import (
	"time"

	"github.com/redis/go-redis/v9"
)

// Service 管理端服务
type Service struct {
	startTime time.Time
	version   string
	env       string
	redis     *redis.Client
}

// NewService 创建管理端服务
func NewService(version, env string, rdb *redis.Client) *Service {
	return &Service{
		startTime: time.Now(),
		version:   version,
		env:       env,
		redis:     rdb,
	}
}

// Version 返回版本号
func (s *Service) Version() string { return s.version }

// Env 返回运行环境
func (s *Service) Env() string { return s.env }
