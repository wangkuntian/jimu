package application

import (
	"context"
	stderrors "errors"
	"testing"

	"jimu/internal/capabilities/product/domain"
	apperrors "jimu/internal/shared/errors"
	"jimu/internal/shared/pagination"

	"gorm.io/gorm"
)

func TestProductServiceListPassesPagination(t *testing.T) {
	seed := []domain.Product{
		{ID: 1, Name: "one"},
	}
	repo := &fakeProductRepository{items: seed, total: 3}
	service := NewProductService(repo)

	items, total, err := service.List(context.Background(), pagination.Pagination{Page: 2, PageSize: 5, Sort: "created_at", Order: "asc"})
	if err != nil {
		t.Fatal(err)
	}
	if repo.offset != 5 || repo.limit != 5 || repo.sort != "created_at" || repo.order != "asc" {
		t.Fatalf("pagination = offset:%d limit:%d sort:%q order:%q", repo.offset, repo.limit, repo.sort, repo.order)
	}
	if total != 3 || len(items) != 1 || items[0].Name != "one" {
		t.Fatalf("items = %#v total = %d", items, total)
	}
}

func TestProductServiceGetMapsNotFound(t *testing.T) {
	service := NewProductService(&fakeProductRepository{findErr: gorm.ErrRecordNotFound})

	_, err := service.Get(context.Background(), 9)
	if appCode(err) != apperrors.CodeNotFound {
		t.Fatalf("code = %d, want %d", appCode(err), apperrors.CodeNotFound)
	}
}

type fakeProductRepository struct {
	item      *domain.Product
	items     []domain.Product
	total     int64
	findErr   error
	createErr error
	updateErr error
	deleteErr error
	offset    int
	limit     int
	sort      string
	order     string
}

func (r *fakeProductRepository) FindByID(context.Context, uint64) (*domain.Product, error) {
	if r.item != nil {
		return r.item, r.findErr
	}
	return &domain.Product{}, r.findErr
}

func (r *fakeProductRepository) List(_ context.Context, offset, limit int, sort, order string) ([]domain.Product, int64, error) {
	r.offset = offset
	r.limit = limit
	r.sort = sort
	r.order = order
	return r.items, r.total, nil
}

func (r *fakeProductRepository) Create(context.Context, *domain.Product) error {
	return r.createErr
}
func (r *fakeProductRepository) Update(context.Context, *domain.Product) error {
	return r.updateErr
}
func (r *fakeProductRepository) Delete(context.Context, uint64) error {
	return r.deleteErr
}

func appCode(err error) int {
	var appErr *apperrors.AppError
	if stderrors.As(err, &appErr) {
		return appErr.Code
	}
	return 0
}
