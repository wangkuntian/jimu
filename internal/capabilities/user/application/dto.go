package application

import (
	"strconv"
	"time"

	"jimu/internal/capabilities/user/domain"
)

type CreateUserRequest struct {
	TenantID string   `json:"tenant_id"`
	Username string   `json:"username" binding:"required,min=3,max=64"`
	Password string   `json:"password" binding:"required,min=6,max=32"`
	Email    string   `json:"email" binding:"omitempty,email"`
	Phone    string   `json:"phone" binding:"omitempty"`
	RoleIDs  []string `json:"role_ids"`
}

type UpdateUserRequest struct {
	Status *int8   `json:"status" binding:"omitempty,oneof=0 1"`
	Email  *string `json:"email" binding:"omitempty,email"`
	Phone  *string `json:"phone" binding:"omitempty"`
}

// BatchDeleteRequest 批量删除请求
type BatchDeleteRequest struct {
	IDs []uint64 `json:"ids" binding:"required,min=1,max=100"`
}

// BatchResult 批量操作结果
type BatchResult struct {
	Success int `json:"success"`
	Failed  int `json:"failed"`
}

type UserResponse struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Phone     string    `json:"phone"`
	Status    int8      `json:"status"`
	Tenant    TenantRef `json:"tenant"`
	Roles     []RoleRef `json:"roles"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type TenantRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type RoleRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func ToUserResponse(user domain.User) UserResponse {
	return UserResponse{
		ID:        strconv.FormatUint(user.ID, 10),
		Username:  user.Username,
		Email:     user.Email,
		Phone:     user.Phone,
		Status:    user.Status,
		Tenant:    TenantRef{ID: strconv.FormatUint(user.TenantID, 10)},
		Roles:     []RoleRef{},
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}

func ToUserResponses(users []domain.User) []UserResponse {
	out := make([]UserResponse, 0, len(users))
	for _, user := range users {
		out = append(out, ToUserResponse(user))
	}
	return out
}

type UserPageResponse struct {
	Code      int            `json:"code"`
	Message   string         `json:"message"`
	Data      []UserResponse `json:"data"`
	Total     int64          `json:"total"`
	Page      int            `json:"page"`
	PageSize  int            `json:"page_size"`
	RequestID string         `json:"request_id,omitempty"`
}

type UserBodyResponse struct {
	Code      int          `json:"code"`
	Message   string       `json:"message"`
	Data      UserResponse `json:"data"`
	RequestID string       `json:"request_id,omitempty"`
}
