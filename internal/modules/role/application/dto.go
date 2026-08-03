package application

import (
	"time"

	"jimu/internal/modules/role/domain"
)

type CreateRoleRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

type UpdateRoleRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

type RoleResponse struct {
	ID          uint64    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      int8      `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type AssignPermissionsRequest struct {
	PermissionIDs []uint64 `json:"permission_ids" binding:"required"`
}

func ToRoleResponse(role domain.Role) RoleResponse {
	return RoleResponse{
		ID:          role.ID,
		Name:        role.Name,
		Description: role.Description,
		Status:      role.Status,
		CreatedAt:   role.CreatedAt,
		UpdatedAt:   role.UpdatedAt,
	}
}

func ToRoleResponses(roles []domain.Role) []RoleResponse {
	out := make([]RoleResponse, 0, len(roles))
	for _, role := range roles {
		out = append(out, ToRoleResponse(role))
	}
	return out
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
