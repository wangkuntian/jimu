package application

import (
	"strconv"
	"time"

	"jimu/internal/capabilities/tenant/domain"
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
	ID        string    `json:"id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	Status    int8      `json:"status"`
	PlanID    uint64    `json:"plan_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func ToTenantResponse(t domain.Tenant) TenantResponse {
	return TenantResponse{
		ID:        strconv.FormatUint(t.ID, 10),
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

// TenantPageResponse 租户列表的分页信封。
// swag 2.0 不支持泛型，故按实体各定义一个具体分页类型（字段与 response.Page 输出一致）。
type TenantPageResponse struct {
	Code      int              `json:"code"`
	Message   string           `json:"message"`
	Data      []TenantResponse `json:"data"`
	Total     int64            `json:"total"`
	Page      int              `json:"page"`
	PageSize  int              `json:"page_size"`
	RequestID string           `json:"request_id,omitempty"`
}

// PlanPageResponse 套餐列表的分页信封。
type PlanPageResponse struct {
	Code      int            `json:"code"`
	Message   string         `json:"message"`
	Data      []PlanResponse `json:"data"`
	Total     int64          `json:"total"`
	Page      int            `json:"page"`
	PageSize  int            `json:"page_size"`
	RequestID string         `json:"request_id,omitempty"`
}

// TenantBodyResponse 单实体租户响应的信封：与 response.Body 的 JSON 形状一致
// （response.OK/Created 写入 code/message/data/request_id）。
type TenantBodyResponse struct {
	Code      int            `json:"code"`
	Message   string         `json:"message"`
	Data      TenantResponse `json:"data"`
	RequestID string         `json:"request_id,omitempty"`
}
