package application

import (
	"strconv"
	"time"

	"jimu/internal/capabilities/access/domain"
)

type CreateRoleRequest struct {
	TenantID    string `json:"tenant_id"`
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

type UpdateRoleRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Status      *int8  `json:"status" binding:"omitempty,oneof=0 1"`
}

type RoleResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      int8      `json:"status"`
	TenantID    string    `json:"tenant_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type RolePageResponse struct {
	Code      int            `json:"code"`
	Message   string         `json:"message"`
	Data      []RoleResponse `json:"data"`
	Total     int64          `json:"total"`
	Page      int            `json:"page"`
	PageSize  int            `json:"page_size"`
	RequestID string         `json:"request_id,omitempty"`
}

type RoleBodyResponse struct {
	Code      int          `json:"code"`
	Message   string       `json:"message"`
	Data      RoleResponse `json:"data"`
	RequestID string       `json:"request_id,omitempty"`
}

type AssignPermissionsRequest struct {
	PermissionIDs *[]string `json:"permission_ids" binding:"required"`
}

type AssignUserRolesRequest struct {
	RoleIDs []string `json:"role_ids"`
}

func ToRoleResponse(role domain.Role) RoleResponse {
	return RoleResponse{
		ID:          strconv.FormatUint(role.ID, 10),
		Name:        role.Name,
		Description: role.Description,
		Status:      role.Status,
		TenantID:    strconv.FormatUint(role.TenantID, 10),
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
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Resource  string    `json:"resource"`
	Action    string    `json:"action"`
	CreatedAt time.Time `json:"created_at"`
}

func ToPermissionResponse(permission domain.Permission) PermissionResponse {
	return PermissionResponse{
		ID:        strconv.FormatUint(permission.ID, 10),
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

type PermissionPageResponse struct {
	Code      int                  `json:"code"`
	Message   string               `json:"message"`
	Data      []PermissionResponse `json:"data"`
	Total     int64                `json:"total"`
	Page      int                  `json:"page"`
	PageSize  int                  `json:"page_size"`
	RequestID string               `json:"request_id,omitempty"`
}

type PermissionListResponse struct {
	Code      int                  `json:"code"`
	Message   string               `json:"message"`
	Data      []PermissionResponse `json:"data"`
	RequestID string               `json:"request_id,omitempty"`
}
