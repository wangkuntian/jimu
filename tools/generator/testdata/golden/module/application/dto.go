package application

import (
	"time"

	"jimu/internal/capabilities/product/domain"
)

type CreateProductRequest struct {
	Name        string `json:"name" binding:"required,max=128"`
	Description string `json:"description" binding:"max=255"`
}

type UpdateProductRequest struct {
	Name        string `json:"name" binding:"required,max=128"`
	Description string `json:"description" binding:"max=255"`
}

type ProductResponse struct {
	ID          uint64    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func ToProductResponse(entity domain.Product) ProductResponse {
	return ProductResponse{
		ID:          entity.ID,
		Name:        entity.Name,
		Description: entity.Description,
		CreatedAt:   entity.CreatedAt,
		UpdatedAt:   entity.UpdatedAt,
	}
}

func ToProductResponses(entities []domain.Product) []ProductResponse {
	out := make([]ProductResponse, 0, len(entities))
	for _, entity := range entities {
		out = append(out, ToProductResponse(entity))
	}
	return out
}
