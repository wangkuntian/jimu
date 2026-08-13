package interfaces

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"jimu/internal/config"
	"jimu/internal/modules/auth/application"
	userdomain "jimu/internal/modules/user/domain"
	"jimu/internal/platform/auth"
	"jimu/internal/platform/encryption"
	"jimu/internal/platform/notification"
	apperrors "jimu/internal/shared/errors"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

const handlerTestKey = "01234567890123456789012345678901"

func TestForgotPasswordHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/forgot-password", func(c *gin.Context) {
		c.Set("validated_req", &forgotPasswordRequest{Email: "user@example.com"})
		NewAuthHandler(newHandlerService(t), config.AuthConfig{}, nil, nil, config.CaptchaConfig{}).ForgotPassword(c)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/forgot-password", strings.NewReader(`{"email":"user@example.com"}`)))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["code"] != float64(0) {
		t.Fatalf("body = %s", w.Body.String())
	}
}

func TestResetPasswordHandlerInvalidCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/reset-password", func(c *gin.Context) {
		c.Set("validated_req", &resetPasswordRequest{Email: "user@example.com", Code: "000000", NewPassword: "newpass123"})
		NewAuthHandler(newHandlerService(t), config.AuthConfig{}, nil, nil, config.CaptchaConfig{}).ResetPassword(c)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/reset-password", strings.NewReader(`{}`)))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["code"] != float64(apperrors.CodeInvalidResetCode) {
		t.Fatalf("body = %s", w.Body.String())
	}
}

// newHandlerService 构造带完整依赖的 AuthService（miniredis 一次性码存储 + fake 仓储/通知）
func newHandlerService(t *testing.T) *application.AuthService {
	t.Helper()
	mrs, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mrs.Close)
	rc := redis.NewClient(&redis.Options{Addr: mrs.Addr()})
	t.Cleanup(func() { _ = rc.Close() })
	return application.NewAuthService(
		&handlerUserRepo{},
		auth.New(handlerTestKey, "jimu", 30, 7),
		&handlerSessionStore{},
		nil, 30,
		encryption.New(handlerTestKey),
		&handlerNotifier{},
		application.NewResetStore(rc, 15*time.Minute),
	)
}

type handlerUserRepo struct{}

func (r *handlerUserRepo) FindByID(context.Context, uint64) (*userdomain.User, error) {
	return nil, gorm.ErrRecordNotFound
}
func (r *handlerUserRepo) FindByUsername(context.Context, string) (*userdomain.User, error) {
	return nil, gorm.ErrRecordNotFound
}
func (r *handlerUserRepo) List(context.Context, int, int, string, string) ([]userdomain.User, int64, error) {
	return nil, 0, nil
}
func (r *handlerUserRepo) Create(context.Context, *userdomain.User) error { return nil }
func (r *handlerUserRepo) Update(context.Context, *userdomain.User) error { return nil }
func (r *handlerUserRepo) Delete(context.Context, uint64) error           { return nil }
func (r *handlerUserRepo) FindByEmailHash(context.Context, string) (*userdomain.User, error) {
	return nil, gorm.ErrRecordNotFound
}
func (r *handlerUserRepo) FindByPhoneHash(context.Context, string) (*userdomain.User, error) {
	return nil, gorm.ErrRecordNotFound
}
func (r *handlerUserRepo) UpdatePassword(context.Context, uint64, string) error { return nil }

type handlerSessionStore struct{}

func (s *handlerSessionStore) Create(context.Context, uint64, string, string, time.Duration) error {
	return nil
}
func (s *handlerSessionStore) Rotate(context.Context, uint64, string, string, string, time.Duration) error {
	return nil
}
func (s *handlerSessionStore) Revoke(context.Context, uint64, string) error { return nil }
func (s *handlerSessionStore) RevokeAll(context.Context, uint64) error      { return nil }

type handlerNotifier struct{}

func (n *handlerNotifier) Register(notification.Channel, notification.Notification) {}
func (n *handlerNotifier) Dispatch(context.Context, notification.Message) error     { return nil }
func (n *handlerNotifier) DispatchBatch(context.Context, []notification.Message) error {
	return nil
}
