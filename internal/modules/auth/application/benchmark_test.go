package application

import (
	"context"
	"testing"

	userdomain "jimu/internal/modules/user/domain"
	"jimu/internal/platform/auth"

	"golang.org/x/crypto/bcrypt"
)

// benchUser 构造带 bcrypt 密码哈希的用户，复用真实登录路径。
func benchUser(id uint64, username, password string) *userdomain.User {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}
	return &userdomain.User{ID: id, Username: username, Password: string(hash), Status: 1}
}

func BenchmarkLogin(b *testing.B) {
	repo := &fakeUserRepo{users: map[string]*userdomain.User{
		"alice": benchUser(42, "alice", "correct"),
	}}
	service := NewAuthService(repo, auth.New("01234567890123456789012345678901", "jimu", 30, 7), newFakeSessionStore(), nil, 30)
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := service.Login(ctx, "alice", "correct"); err != nil {
			b.Fatal(err)
		}
	}
}
