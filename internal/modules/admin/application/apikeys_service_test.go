package application

import (
	"context"
	"errors"
	"testing"
	"time"

	admindomain "jimu/internal/modules/admin/domain"

	"github.com/stretchr/testify/assert"
)

func TestAdminAPIKeyServiceCreateKey(t *testing.T) {
	ctx := context.Background()

	// 名为空
	svc := NewAdminAPIKeyService(&fakeAPIKeyRepo{})
	_, _, err := svc.CreateKey(ctx, CreateKeyInput{})
	assert.Error(t, err)

	// 成功：无过期时间
	svc = NewAdminAPIKeyService(&fakeAPIKeyRepo{})
	plain, key, err := svc.CreateKey(ctx, CreateKeyInput{Name: "web", Scopes: []string{"read", "write"}, CreatedBy: 1})
	assert.NoError(t, err)
	assert.NotEmpty(t, plain)
	assert.True(t, len(plain) > len(apiKeyPrefix))
	assert.True(t, plain[:len(apiKeyPrefix)] == apiKeyPrefix)
	assert.Equal(t, admindomain.HashKey(plain), key.KeyHash)
	assert.Equal(t, "[\"read\",\"write\"]", key.Scopes)
	assert.True(t, key.Enabled)
	assert.True(t, key.ExpiresAt.IsZero())

	// 成功：带过期时间
	svc = NewAdminAPIKeyService(&fakeAPIKeyRepo{})
	_, key, err = svc.CreateKey(ctx, CreateKeyInput{Name: "web", ExpiresIn: 30})
	assert.NoError(t, err)
	assert.False(t, key.ExpiresAt.IsZero())
	expected := time.Now().Add(30 * 24 * time.Hour)
	assert.WithinDuration(t, expected, key.ExpiresAt, time.Minute)

	// 仓储错误
	svc = NewAdminAPIKeyService(&fakeAPIKeyRepo{create: func(ctx context.Context, key *admindomain.APIKey) error {
		return errors.New("db down")
	}})
	_, _, err = svc.CreateKey(ctx, CreateKeyInput{Name: "web"})
	assert.Error(t, err)
}

func TestAdminAPIKeyServiceListKeys(t *testing.T) {
	ctx := context.Background()
	svc := NewAdminAPIKeyService(&fakeAPIKeyRepo{})
	keys, total, err := svc.ListKeys(ctx, 0, 20)
	assert.NoError(t, err)
	assert.Len(t, keys, 1)
	assert.Equal(t, int64(1), total)

	// 错误
	svc = NewAdminAPIKeyService(&fakeAPIKeyRepo{list: func(ctx context.Context, offset, limit int) ([]admindomain.APIKey, int64, error) {
		return nil, 0, errors.New("db down")
	}})
	_, _, err = svc.ListKeys(ctx, 0, 20)
	assert.Error(t, err)
}

func TestAdminAPIKeyServiceGetKey(t *testing.T) {
	ctx := context.Background()
	svc := NewAdminAPIKeyService(&fakeAPIKeyRepo{})
	key, err := svc.GetKey(ctx, 9)
	assert.NoError(t, err)
	assert.Equal(t, uint64(9), key.ID)

	svc = NewAdminAPIKeyService(&fakeAPIKeyRepo{findByID: func(ctx context.Context, id uint64) (*admindomain.APIKey, error) {
		return nil, errors.New("not found")
	}})
	_, err = svc.GetKey(ctx, 9)
	assert.Error(t, err)
}

func TestAdminAPIKeyServiceRevokeKey(t *testing.T) {
	ctx := context.Background()
	svc := NewAdminAPIKeyService(&fakeAPIKeyRepo{})
	assert.NoError(t, svc.RevokeKey(ctx, 9))

	svc = NewAdminAPIKeyService(&fakeAPIKeyRepo{delete: func(ctx context.Context, id uint64) error {
		return errors.New("db down")
	}})
	assert.Error(t, svc.RevokeKey(ctx, 9))
}
