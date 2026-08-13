package application

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	userdomain "jimu/internal/modules/user/domain"
	"jimu/internal/platform/auth"
	"jimu/internal/platform/encryption"
	"jimu/internal/platform/notification"
	apperrors "jimu/internal/shared/errors"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

const resetTestKey = "0123456789abcdef0123456789abcdef"

// newResetRedis 启动内存 redis（miniredis）供一次性码存储测试
func newResetRedis(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	t.Helper()
	mrs, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mrs.Close)
	client := redis.NewClient(&redis.Options{Addr: mrs.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return mrs, client
}

// fakeDispatcher 记录通知消息的 Dispatcher mock
type fakeDispatcher struct {
	mu       sync.Mutex
	messages []notification.Message
}

func (f *fakeDispatcher) Register(notification.Channel, notification.Notification) {}
func (f *fakeDispatcher) Dispatch(_ context.Context, msg notification.Message) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.messages = append(f.messages, msg)
	return nil
}
func (f *fakeDispatcher) DispatchBatch(_ context.Context, msgs []notification.Message) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.messages = append(f.messages, msgs...)
	return nil
}

func (f *fakeDispatcher) sent() []notification.Message {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]notification.Message{}, f.messages...)
}

func newResetService(t *testing.T, repo *fakeUserRepo, notifier *fakeDispatcher) (*AuthService, *ResetStore, *fakeSessionStore) {
	t.Helper()
	_, rclient := newResetRedis(t)
	resetStore := NewResetStore(rclient, 15*time.Minute)
	store := newFakeSessionStore()
	svc := NewAuthService(repo, auth.New("01234567890123456789012345678901", "jimu", 30, 7), store, nil, 30,
		encryption.New(resetTestKey), notifier, resetStore)
	svc.resetGen = func() string { return "123456" }
	return svc, resetStore, store
}

func TestForgotPasswordSendsCode(t *testing.T) {
	ctx := context.Background()
	repo := &fakeUserRepo{users: map[string]*userdomain.User{
		"alice": userWithPassword(t, 42, "alice", "correct", 1),
	}}
	repo.findByEmailHash = func(_ context.Context, hash string) (*userdomain.User, error) {
		return repo.users["alice"], nil
	}
	notifier := &fakeDispatcher{}
	svc, resetStore, _ := newResetService(t, repo, notifier)

	if err := svc.ForgotPassword(ctx, "alice@example.com"); err != nil {
		t.Fatal(err)
	}
	msgs := notifier.sent()
	if len(msgs) != 1 || msgs[0].To != "alice@example.com" || msgs[0].Channel != notification.ChannelEmail {
		t.Fatalf("messages = %#v", msgs)
	}
	if !strings.Contains(msgs[0].Body, "123456") {
		t.Fatalf("body lacks code: %q", msgs[0].Body)
	}
	code, err := resetStore.GetAndDelete(ctx, encryption.New(resetTestKey).BlindIndex("alice@example.com"))
	if err != nil || code != "123456" {
		t.Fatalf("stored code = %q err = %v", code, err)
	}
}

func TestForgotPasswordHidesMissingUser(t *testing.T) {
	ctx := context.Background()
	notifier := &fakeDispatcher{}
	svc, _, _ := newResetService(t, &fakeUserRepo{users: map[string]*userdomain.User{}}, notifier)

	if err := svc.ForgotPassword(ctx, "missing@example.com"); err != nil {
		t.Fatalf("missing user must not error: %v", err)
	}
	if len(notifier.sent()) != 0 {
		t.Fatalf("must not send to missing user: %#v", notifier.sent())
	}
}

func TestResetPasswordSuccess(t *testing.T) {
	ctx := context.Background()
	repo := &fakeUserRepo{users: map[string]*userdomain.User{
		"alice": userWithPassword(t, 42, "alice", "correct", 1),
	}}
	var updatedID uint64
	var updatedHash string
	repo.findByEmailHash = func(_ context.Context, hash string) (*userdomain.User, error) {
		return repo.users["alice"], nil
	}
	repo.updatePassword = func(_ context.Context, id uint64, hashed string) error {
		updatedID, updatedHash = id, hashed
		return nil
	}
	notifier := &fakeDispatcher{}
	svc, resetStore, store := newResetService(t, repo, notifier)
	hash := encryption.New(resetTestKey).BlindIndex("alice@example.com")
	require.NoError(t, resetStore.Set(ctx, hash, "123456"))

	if err := svc.ResetPassword(ctx, "alice@example.com", "123456", "newpass123"); err != nil {
		t.Fatal(err)
	}
	if updatedID != 42 || updatedHash == "" {
		t.Fatalf("password not updated: id=%d hash=%q", updatedID, updatedHash)
	}
	if len(store.revokedAll) != 1 || store.revokedAll[0] != 42 {
		t.Fatalf("revokedAll = %#v", store.revokedAll)
	}
}

func TestResetPasswordWrongCode(t *testing.T) {
	ctx := context.Background()
	repo := &fakeUserRepo{users: map[string]*userdomain.User{
		"alice": userWithPassword(t, 42, "alice", "correct", 1),
	}}
	notifier := &fakeDispatcher{}
	svc, resetStore, _ := newResetService(t, repo, notifier)
	require.NoError(t, resetStore.Set(ctx, encryption.New(resetTestKey).BlindIndex("alice@example.com"), "123456"))

	err := svc.ResetPassword(ctx, "alice@example.com", "000000", "newpass123")
	if appCode(err) != apperrors.CodeInvalidResetCode {
		t.Fatalf("code = %d, want %d", appCode(err), apperrors.CodeInvalidResetCode)
	}
}

func TestForgotPasswordNotConfigured(t *testing.T) {
	ctx := context.Background()
	svc := NewAuthService(&fakeUserRepo{users: map[string]*userdomain.User{}}, auth.New(resetTestKey, "jimu", 30, 7), newFakeSessionStore(), nil, 30)
	err := svc.ForgotPassword(ctx, "a@example.com")
	if appCode(err) != apperrors.CodeInternalError {
		t.Fatalf("code = %d, want %d", appCode(err), apperrors.CodeInternalError)
	}
}

func TestGenerateResetCodeReal(t *testing.T) {
	repo := &fakeUserRepo{users: map[string]*userdomain.User{}}
	svc, _, _ := newResetService(t, repo, &fakeDispatcher{})
	svc.resetGen = nil // 走真实 crypto/rand 路径
	for i := 0; i < 5; i++ {
		code := svc.generateResetCode()
		if len(code) != 6 {
			t.Fatalf("code = %q, want 6 digits", code)
		}
		for _, r := range code {
			if r < '0' || r > '9' {
				t.Fatalf("code = %q contains non-digit", code)
			}
		}
	}
}

func TestResetPasswordCodeReuseFails(t *testing.T) {
	ctx := context.Background()
	repo := &fakeUserRepo{users: map[string]*userdomain.User{
		"alice": userWithPassword(t, 42, "alice", "correct", 1),
	}}
	repo.findByEmailHash = func(_ context.Context, hash string) (*userdomain.User, error) {
		return repo.users["alice"], nil
	}
	notifier := &fakeDispatcher{}
	svc, resetStore, _ := newResetService(t, repo, notifier)
	hash := encryption.New(resetTestKey).BlindIndex("alice@example.com")
	require.NoError(t, resetStore.Set(ctx, hash, "123456"))

	if err := svc.ResetPassword(ctx, "alice@example.com", "123456", "newpass123"); err != nil {
		t.Fatal(err)
	}
	// 码已被原子消费，重用必须失败
	if err := svc.ResetPassword(ctx, "alice@example.com", "123456", "again123"); appCode(err) != apperrors.CodeInvalidResetCode {
		t.Fatalf("reuse code = %d, want %d", appCode(err), apperrors.CodeInvalidResetCode)
	}
}
