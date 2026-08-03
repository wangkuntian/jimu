package application

import (
	"time"

	"jimu/internal/modules/role/domain"
)

type CreatePermissionRequest struct {
	Name     string `json:"name" binding:"required"`
	Resource string `json:"resource" binding:"required"`
	Action   string `json:"action" binding:"required"`
}

type UpdatePermissionRequest struct {
	Name     string `json:"name" binding:"required"`
	Resource string `json:"resource" binding:"required"`
	Action   string `json:"action" binding:"required"`
}

type PermissionResponse struct {
	ID        uint64    `json:"id"`
	Name      string    `json:"name"`
	Resource  string    `json:"resource"`
	Action    string    `json:"action"`
	CreatedAt time.Time `json:"created_at"`
}

func ToPermissionResponse(permission domain.Permission) PermissionResponse {
	return PermissionResponse{
		ID:        permission.ID,
		Name:      permission.Name,
		Resource:  permission.Resource,
		Action:    permission.Action,
		CreatedAt: permission.CreatedAt,
	}
}

func ToPermissionResponses(permissions []domain.Permission) []PermissionResponse {
	out := make([]PermissionResponse, 0, len(permissions))
	for _, permission := range permissions {
		out = append(out, ToPermissionResponse(permission))
	}
	return out
}
