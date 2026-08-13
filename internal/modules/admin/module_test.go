package admin

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"jimu/internal/platform/auth"
	"jimu/internal/platform/feature"
	"jimu/internal/platform/scheduler"

	"github.com/stretchr/testify/assert"
)

type fakeEventBus struct{}

func (fakeEventBus) Subscribe(event string, handler func(payload interface{})) {}
func (fakeEventBus) Publish(event string, payload interface{})                 {}

// fakeStorage 存储抽象测试桩
type fakeStorage struct{}

func (fakeStorage) Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error {
	return nil
}
func (fakeStorage) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	return nil, nil
}
func (fakeStorage) Delete(ctx context.Context, key string) error { return nil }
func (fakeStorage) Exists(ctx context.Context, key string) (bool, error) {
	return false, nil
}
func (fakeStorage) Size(ctx context.Context, key string) (int64, error) { return 0, nil }
func (fakeStorage) URL(key string) string                               { return "" }
func (fakeStorage) PresignedURL(key string, expiry time.Duration) (string, error) {
	return "", nil
}
func (fakeStorage) PresignedUploadURL(key string, expiry time.Duration, contentType string) (string, error) {
	return "", nil
}

func TestModuleNameAndNew(t *testing.T) {
	m := New("v1", "test", nil, nil)
	assert.Equal(t, "admin", m.Name())

	// deps 注入覆盖 New 的 switch 分支
	m2 := New("v1", "test", nil, nil,
		&scheduler.CronScheduler{},
		fakeStorage{},
		feature.NewManager(),
		fakeEventBus{},
		&auth.JWT{},
	)
	assert.NotNil(t, m2.sched)
	assert.NotNil(t, m2.storage)
	assert.NotNil(t, m2.feature)
	assert.NotNil(t, m2.eventBus)
	assert.NotNil(t, m2.jwt)

	// 无 storage / feature 时对应字段保持 nil
	m3 := New("v1", "test", nil, nil)
	assert.Nil(t, m3.storage)
	assert.Nil(t, m3.feature)
}

func TestModuleInitWSIdempotent(t *testing.T) {
	m := &Module{}
	m.initWS()
	assert.NotNil(t, m.wsHub)
	assert.NotNil(t, m.wsPres)
	first := m.wsHub
	m.initWS()
	assert.Same(t, first, m.wsHub)
}

func TestModuleWSHandler(t *testing.T) {
	// 未注入 JWT -> 返回 nil
	m := &Module{}
	assert.Nil(t, m.wsHandler())

	// 注入 JWT -> 返回可用的 handler
	m2 := New("v1", "test", nil, nil, auth.New("secret", "jimu", 60, 7))
	h := m2.wsHandler()
	assert.NotNil(t, h)
	// 缺少 token 返回 401
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	w := httptest.NewRecorder()
	h(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
