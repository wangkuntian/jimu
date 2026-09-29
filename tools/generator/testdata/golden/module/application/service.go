package application

import (
	"context"
	stderrors "errors"

	"jimu/internal/capabilities/product/domain"
	"jimu/internal/shared/errors"
	"jimu/internal/shared/pagination"

	"gorm.io/gorm"
)

type ProductService struct {
	repo domain.ProductRepository
}

func NewProductService(repo domain.ProductRepository) *ProductService {
	return &ProductService{repo: repo}
}

func (s *ProductService) Create(ctx context.Context, req CreateProductRequest) (*ProductResponse, error) {
	entity := &domain.Product{Name: req.Name, Description: req.Description}
	if err := s.repo.Create(ctx, entity); err != nil {
		if isDuplicateKey(err) {
			return nil, errors.Wrap(errors.CodeConflict, "product already exists", err)
		}
		return nil, errors.Wrap(errors.CodeInternalError, "failed to create product", err)
	}
	resp := ToProductResponse(*entity)
	return &resp, nil
}

func (s *ProductService) Get(ctx context.Context, id uint64) (*ProductResponse, error) {
	entity, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.Wrap(errors.CodeNotFound, "product not found", err)
		}
		return nil, errors.Wrap(errors.CodeInternalError, "failed to get product", err)
	}
	resp := ToProductResponse(*entity)
	return &resp, nil
}

func (s *ProductService) List(ctx context.Context, p pagination.Pagination) ([]ProductResponse, int64, error) {
	entities, total, err := s.repo.List(ctx, p.GetOffset(), p.GetLimit(), p.Sort, p.Order)
	if err != nil {
		return nil, 0, errors.Wrap(errors.CodeInternalError, "failed to list product", err)
	}
	return ToProductResponses(entities), total, nil
}

func (s *ProductService) Update(ctx context.Context, id uint64, req UpdateProductRequest) error {
	entity, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return errors.Wrap(errors.CodeNotFound, "product not found", err)
		}
		return errors.Wrap(errors.CodeInternalError, "failed to get product", err)
	}
	entity.Name = req.Name
	entity.Description = req.Description
	if err := s.repo.Update(ctx, entity); err != nil {
		if isDuplicateKey(err) {
			return errors.Wrap(errors.CodeConflict, "product already exists", err)
		}
		return errors.Wrap(errors.CodeInternalError, "failed to update product", err)
	}
	return nil
}

func (s *ProductService) Delete(ctx context.Context, id uint64) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return errors.Wrap(errors.CodeInternalError, "failed to delete product", err)
	}
	return nil
}
