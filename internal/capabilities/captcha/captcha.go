package captcha

import (
	"context"
	"fmt"
	"time"

	"jimu/internal/contract"

	"github.com/mojocn/base64Captcha"
)

// Store 是验证码存储：实现 base64Captcha.Store 接口，底层用 cache 端口。
//
// 不直接依赖 Redis：缓存能力已把「跨实例」与「进程内」抽象成驱动，
// 验证码只需要 Get/Set/Delete 三种语义。
type Store struct {
	cache contract.Cache
	ttl   time.Duration
}

// NewStore 创建验证码存储。
func NewStore(cache contract.Cache, ttl time.Duration) *Store {
	return &Store{cache: cache, ttl: ttl}
}

// Set 存储验证码，写入 jimu:captcha:{id}，带 TTL
func (s *Store) Set(id, value string) error {
	key := fmt.Sprintf("jimu:captcha:%s", id)
	return s.cache.Set(context.Background(), key, value, s.ttl)
}

// Get 读取验证码，clear 为 true 时读取后删除（一次性）
func (s *Store) Get(id string, clear bool) string {
	key := fmt.Sprintf("jimu:captcha:%s", id)
	var stored string
	found, err := s.cache.Get(context.Background(), key, &stored)
	if err != nil || !found {
		return ""
	}
	if clear {
		_ = s.cache.Delete(context.Background(), key)
	}
	return stored
}

// Verify 校验验证码，clear 为 true 时无论成败都删除
func (s *Store) Verify(id, answer string, clear bool) bool {
	return s.Get(id, clear) == answer
}

// Service 验证码服务（生成 + 缓存存储 + 校验）
type Service struct {
	store   *Store
	enabled bool
}

// NewService 创建验证码服务（默认启用）。
func NewService(cache contract.Cache, ttl time.Duration) *Service {
	return &Service{
		store:   NewStore(cache, ttl),
		enabled: true,
	}
}

// NewServiceWithEnabled 创建验证码服务并显式指定启用状态。
// enabled=false 时 Verify 直接放行（未启用验证码的站点不校验）。
func NewServiceWithEnabled(cache contract.Cache, ttl time.Duration, enabled bool) *Service {
	return &Service{
		store:   NewStore(cache, ttl),
		enabled: enabled,
	}
}

// Enabled 是否启用验证码校验（实现 contract.CaptchaVerifier）。
func (s *Service) Enabled() bool { return s.enabled }

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
	if !s.store.Verify(id, code, true) {
		return fmt.Errorf("invalid captcha")
	}
	return nil
}
