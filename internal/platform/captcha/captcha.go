package captcha

import (
	"context"
	"fmt"
	"time"

	"github.com/mojocn/base64Captcha"
	"github.com/redis/go-redis/v9"
)

// RedisStore 基于 go-redis 的验证码存储，实现 base64Captcha.Store 接口
type RedisStore struct {
	client *redis.Client
	ttl    time.Duration
}

// NewRedisStore 创建 Redis 验证码存储
func NewRedisStore(client *redis.Client, ttl time.Duration) *RedisStore {
	return &RedisStore{client: client, ttl: ttl}
}

// Set 存储验证码，写入 jimu:captcha:{id}，带 TTL
func (s *RedisStore) Set(id, value string) error {
	key := fmt.Sprintf("jimu:captcha:%s", id)
	return s.client.Set(context.Background(), key, value, s.ttl).Err()
}

// Get 读取验证码，clear 为 true 时读取后删除（一次性）
func (s *RedisStore) Get(id string, clear bool) string {
	key := fmt.Sprintf("jimu:captcha:%s", id)
	stored, err := s.client.Get(context.Background(), key).Result()
	if err != nil {
		return ""
	}
	if clear {
		_ = s.client.Del(context.Background(), key).Err()
	}
	return stored
}

// Verify 校验验证码，clear 为 true 时无论成败都删除
func (s *RedisStore) Verify(id, answer string, clear bool) bool {
	return s.Get(id, clear) == answer
}

// Service 验证码服务（生成 + Redis 存储 + 校验）
type Service struct {
	client *redis.Client
	store  *RedisStore
	ttl    time.Duration
}

// NewService 创建验证码服务
func NewService(client *redis.Client, ttl time.Duration) *Service {
	return &Service{
		client: client,
		store:  NewRedisStore(client, ttl),
		ttl:    ttl,
	}
}

// Generate 生成验证码，返回 id 与 base64 图片
func (s *Service) Generate(ctx context.Context) (string, string, error) {
	driver := base64Captcha.NewDriverDigit(80, 240, 4, 0.7, 80)
	captcha := base64Captcha.NewCaptcha(driver, s.store)
	id, b64, _, err := captcha.Generate()
	if err != nil {
		return "", "", fmt.Errorf("generate captcha: %w", err)
	}
	return id, b64, nil
}

// Verify 校验验证码，校验后无论成败都删除（一次性）
func (s *Service) Verify(ctx context.Context, id, code string) error {
	key := fmt.Sprintf("jimu:captcha:%s", id)
	stored, err := s.client.Get(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("captcha not found or expired")
	}
	_ = s.client.Del(ctx, key).Err()
	if stored != code {
		return fmt.Errorf("captcha code mismatch")
	}
	return nil
}
