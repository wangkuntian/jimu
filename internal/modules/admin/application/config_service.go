package application

import (
	"context"
	"fmt"
	"strings"

	"jimu/internal/contract"

	redistore "jimu/internal/platform/redis"
)

// AdminConfigService 配置热更新服务
type AdminConfigService struct {
	redis    redistore.Client
	eventBus contract.EventBus
	prefix   string
}

// NewAdminConfigService 创建配置热更新服务
func NewAdminConfigService(rdb redistore.Client, eb contract.EventBus, prefix string) *AdminConfigService {
	return &AdminConfigService{redis: rdb, eventBus: eb, prefix: prefix}
}

// GetAllConfig 获取所有配置
func (s *AdminConfigService) GetAllConfig(ctx context.Context) (map[string]string, error) {
	pattern := s.configKey("*")
	iter := s.redis.Scan(ctx, 0, pattern, 0).Iterator()
	result := map[string]string{}
	for iter.Next(ctx) {
		key := iter.Val()
		val, err := s.redis.Get(ctx, key).Result()
		if err == nil {
			shortKey := strings.TrimPrefix(key, s.configKey("")) // strip "jimu:config:"
			result[shortKey] = val
		}
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// UpdateConfig 更新配置（Redis + Event Bus 多节点同步）
func (s *AdminConfigService) UpdateConfig(ctx context.Context, key, value string) error {
	if err := s.redis.Set(ctx, s.configKey(key), value, 0).Err(); err != nil {
		return err
	}
	// 发布事件通知所有节点刷新本地缓存
	s.eventBus.Publish("config.updated", map[string]string{"key": key, "value": value})
	return nil
}

// ReloadConfig 从 Redis 重读全部配置并发布事件，触发各节点应用
func (s *AdminConfigService) ReloadConfig(ctx context.Context) error {
	all, err := s.GetAllConfig(ctx)
	if err != nil {
		return err
	}
	for key, value := range all {
		s.eventBus.Publish("config.updated", map[string]string{"key": key, "value": value})
	}
	return nil
}

// IsValidKey 验证配置 key 是否合法
func (s *AdminConfigService) IsValidKey(key string) bool {
	allowed := map[string]bool{
		"rate_limit_rate":  true,
		"rate_limit_burst": true,
		"log_level":        true,
		"feature_flags":    true,
	}
	return allowed[key]
}

func (s *AdminConfigService) configKey(key string) string {
	return fmt.Sprintf("%s:config:%s", s.prefix, key)
}
