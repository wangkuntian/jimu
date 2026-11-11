package captcha

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// fakeCache 是 contract.Cache 的最小内存实现：验证码测试只关心 Get/Set/Delete。
type fakeCache struct {
	mu    sync.Mutex
	items map[string]string
}

func newFakeCache() *fakeCache { return &fakeCache{items: map[string]string{}} }

func (c *fakeCache) Get(_ context.Context, key string, dest any) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	value, ok := c.items[key]
	if !ok {
		return false, nil
	}
	target, ok := dest.(*string)
	if !ok {
		return false, nil
	}
	*target = value
	return true, nil
}

func (c *fakeCache) Set(_ context.Context, key string, value any, _ time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	text, _ := value.(string)
	c.items[key] = text
	return nil
}

func (c *fakeCache) Delete(_ context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
	return nil
}

func TestGenerate(t *testing.T) {
	s := NewService(newFakeCache(), 5*time.Minute)

	id, img, err := s.Generate(context.Background())
	assert.NoError(t, err)
	assert.NotEmpty(t, id)
	assert.Contains(t, img, "data:image")
}

func TestVerify(t *testing.T) {
	cache := newFakeCache()
	s := NewService(cache, 5*time.Minute)

	// 直接写入一个验证码验证校验逻辑
	err := cache.Set(context.Background(), "jimu:captcha:test123", "1234", 5*time.Minute)
	assert.NoError(t, err)

	err = s.Verify(context.Background(), "test123", "1234")
	assert.NoError(t, err)

	// 一次性：再次校验失败
	err = s.Verify(context.Background(), "test123", "1234")
	assert.Error(t, err)
}
