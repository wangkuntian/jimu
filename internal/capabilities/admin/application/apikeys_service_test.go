package application

import (
	"context"
	"errors"
	"testing"
	"time"

	admindomain "jimu/internal/capabilities/admin/domain"
	"jimu/internal/platform/tenant"

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
	svc = NewAdminAPIKeyService(&fakeAPIKeyRepo{list: func(ctx context.Context, tenantID uint64, offset, limit int) ([]admindomain.APIKey, int64, error) {
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

func TestAdminAPIKeyServiceTenantBinding(t *testing.T) {
	// 创建：归属上下文租户
	var created *admindomain.APIKey
	svc := NewAdminAPIKeyService(&fakeAPIKeyRepo{create: func(ctx context.Context, key *admindomain.APIKey) error {
		created = key
		return nil
	}})
	_, _, err := svc.CreateKey(tenant.WithTenant(context.Background(), 7), CreateKeyInput{Name: "web"})
	assert.NoError(t, err)
	assert.Equal(t, uint64(7), created.TenantID)

	// 创建：上下文无租户时归默认租户
	_, _, err = svc.CreateKey(context.Background(), CreateKeyInput{Name: "web"})
	assert.NoError(t, err)
	assert.Equal(t, tenant.DefaultTenantID, created.TenantID)

	// 列表：上下文租户透传仓储（0=平台级视角）
	var gotTenant uint64
	svc = NewAdminAPIKeyService(&fakeAPIKeyRepo{list: func(ctx context.Context, tenantID uint64, offset, limit int) ([]admindomain.APIKey, int64, error) {
		gotTenant = tenantID
		return nil, 0, nil
	}})
	_, _, err = svc.ListKeys(tenant.WithTenant(context.Background(), 7), 0, 20)
	assert.NoError(t, err)
	assert.Equal(t, uint64(7), gotTenant)

	// 详情/撤销：跨租户不可见
	svc = NewAdminAPIKeyService(&fakeAPIKeyRepo{findByID: func(ctx context.Context, id uint64) (*admindomain.APIKey, error) {
		return &admindomain.APIKey{ID: id, TenantID: 2}, nil
	}})
	_, err = svc.GetKey(tenant.WithTenant(context.Background(), 7), 1)
	assert.Error(t, err)
	assert.Error(t, svc.RevokeKey(tenant.WithTenant(context.Background(), 7), 1))

	// 同租户可见
	_, err = svc.GetKey(tenant.WithTenant(context.Background(), 2), 1)
	assert.NoError(t, err)
	assert.NoError(t, svc.RevokeKey(tenant.WithTenant(context.Background(), 2), 1))
}
