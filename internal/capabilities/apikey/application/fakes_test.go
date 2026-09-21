package application

import (
	"context"

	apikeydomain "jimu/internal/capabilities/apikey/domain"

	"gorm.io/gorm"
)

// fakeAPIKeyRepo 可配置的 API Key 仓储 mock
type fakeAPIKeyRepo struct {
	create   func(ctx context.Context, key *apikeydomain.APIKey) error
	findByID func(ctx context.Context, id uint64) (*apikeydomain.APIKey, error)
	list     func(ctx context.Context, tenantID uint64, offset, limit int) ([]apikeydomain.APIKey, int64, error)
	delete   func(ctx context.Context, id uint64) error
}

func (f *fakeAPIKeyRepo) Create(ctx context.Context, key *apikeydomain.APIKey) error {
	if f.create != nil {
		return f.create(ctx, key)
	}
	key.ID = 9
	return nil
}

func (f *fakeAPIKeyRepo) FindByID(ctx context.Context, id uint64) (*apikeydomain.APIKey, error) {
	if f.findByID != nil {
		return f.findByID(ctx, id)
	}
	return &apikeydomain.APIKey{ID: id, Name: "web"}, nil
}

func (f *fakeAPIKeyRepo) FindByKeyHash(ctx context.Context, hash string) (*apikeydomain.APIKey, error) {
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeAPIKeyRepo) List(ctx context.Context, tenantID uint64, offset, limit int) ([]apikeydomain.APIKey, int64, error) {
	if f.list != nil {
		return f.list(ctx, tenantID, offset, limit)
	}
	return []apikeydomain.APIKey{{ID: 1, Name: "web"}}, 1, nil
}

func (f *fakeAPIKeyRepo) Update(ctx context.Context, key *apikeydomain.APIKey) error { return nil }
func (f *fakeAPIKeyRepo) Delete(ctx context.Context, id uint64) error {
	if f.delete != nil {
		return f.delete(ctx, id)
	}
	return nil
}

func (f *fakeAPIKeyRepo) IncrementUseCount(ctx context.Context, id uint64) error { return nil }
