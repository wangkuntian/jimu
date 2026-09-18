package captcha

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func TestGenerate(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	s := NewService(client, 5*time.Minute)

	id, img, err := s.Generate(context.Background())
	assert.NoError(t, err)
	assert.NotEmpty(t, id)
	assert.Contains(t, img, "data:image")
}

func TestVerify(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	s := NewService(client, 5*time.Minute)

	// 直接写入一个验证码验证校验逻辑
	key := "jimu:captcha:test123"
	_ = client.Set(context.Background(), key, "1234", 5*time.Minute).Err()

	err := s.Verify(context.Background(), "test123", "1234")
	assert.NoError(t, err)

	// 一次性：再次校验失败
	err = s.Verify(context.Background(), "test123", "1234")
	assert.Error(t, err)
}
