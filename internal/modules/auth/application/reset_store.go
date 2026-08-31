package application

import (
	"context"
	"fmt"
	"time"

	redistore "jimu/internal/platform/redis"

	"github.com/redis/go-redis/v9"
)

// resetKeyPrefix 密码重置验证码 key 前缀（后接邮箱盲索引）
const resetKeyPrefix = "jimu:reset:"

// ResetStore 密码重置验证码存储（Redis 一次性码，key 为邮箱盲索引）
type ResetStore struct {
	client redistore.Client
	ttl    time.Duration
}

// NewResetStore 创建密码重置验证码存储
func NewResetStore(client redistore.Client, ttl time.Duration) *ResetStore {
	return &ResetStore{client: client, ttl: ttl}
}

// Set 存储验证码，TTL 同 call
func (s *ResetStore) Set(ctx context.Context, emailHash, code string) error {
	key := fmt.Sprintf("%s%s", resetKeyPrefix, emailHash)
	return s.client.Set(ctx, key, code, s.ttl).Err()
}

// GetAndDelete 原子读取并删除验证码（Lua 保证并发下一次有效，一次性）
func (s *ResetStore) GetAndDelete(ctx context.Context, emailHash string) (string, error) {
	key := fmt.Sprintf("%s%s", resetKeyPrefix, emailHash)
	script := redis.NewScript(`
local v = redis.call("GET", KEYS[1])
if v then redis.call("DEL", KEYS[1]) end
return v`)
	val, err := script.Run(ctx, s.client, []string{key}).Result()
	if err != nil {
		if err == redis.Nil {
			return "", nil
		}
		return "", err
	}
	if val == nil {
		return "", nil
	}
	code, _ := val.(string)
	return code, nil
}
