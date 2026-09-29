package domain

import "context"

type ProductRepository interface {
	FindByID(ctx context.Context, id uint64) (*Product, error)
	List(ctx context.Context, offset, limit int, sort, order string) ([]Product, int64, error)
	Create(ctx context.Context, entity *Product) error
	Update(ctx context.Context, entity *Product) error
	Delete(ctx context.Context, id uint64) error
}
