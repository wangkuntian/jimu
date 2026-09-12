package application

import (
	"time"

	"jimu/internal/modules/tenant/domain"
)

type CreateTenantRequest struct {
	Code string `json:"code" binding:"required,max=64"`
	Name string `json:"name" binding:"required,min=1,max=128"`
}

type UpdateTenantRequest struct {
	Name   *string `json:"name" binding:"omitempty,min=1,max=128"`
	Status *int8   `json:"status" binding:"omitempty,oneof=0 1"`
}

type TenantResponse struct {
	ID        uint64    `json:"id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	Status    int8      `json:"status"`
	PlanID    uint64    `json:"plan_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func ToTenantResponse(t domain.Tenant) TenantResponse {
	return TenantResponse{
		ID:        t.ID,
		Code:      t.Code,
		Name:      t.Name,
		Status:    t.Status,
		PlanID:    t.PlanID,
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
	}
}

func ToTenantResponses(tenants []domain.Tenant) []TenantResponse {
	out := make([]TenantResponse, 0, len(tenants))
	for _, t := range tenants {
		out = append(out, ToTenantResponse(t))
	}
	return out
}
