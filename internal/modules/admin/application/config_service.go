package application

import (
	"context"
	"encoding/json"
	"fmt"

	"jimu/internal/platform/event"

	"github.com/redis/go-redis/v9"
)

// AdminConfigService 配置热更新服务
type AdminConfigService struct {
	redis    *redis.Client
	eventBus *event.EventBus
	prefix   string
}

// NewAdminConfigService 创建配置热更新服务
func NewAdminConfigService(rdb *redis.Client, eb *event.EventBus, prefix string) *AdminConfigService {
	return &AdminConfigService{redis: rdb, eventBus: eb, prefix: prefix}
}

// GetConfig 获取单个配置
func (s *AdminConfigService) GetConfig(ctx context.Context, key string) (string, error) {
	val, err := s.redis.Get(ctx, s.configKey(key)).Result()
	if err != nil {
		return "", err
	}
	return val, nil
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
			shortKey := key[len(s.prefix)+1:] // strip "jimu:config:"
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
	return fmt.Sprintf("jimu:config:%s", key)
}

// ToJSON JSON 序列化辅助
func (s *AdminConfigService) ToJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}
