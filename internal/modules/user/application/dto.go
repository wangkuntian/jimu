package application

import (
	"time"

	"jimu/internal/modules/user/domain"
)

type CreateUserRequest struct {
	Username string `json:"username" binding:"required,min=3,max=64"`
	Password string `json:"password" binding:"required,min=6,max=32"`
	Email    string `json:"email" binding:"omitempty,email"`
	Phone    string `json:"phone" binding:"omitempty"`
}

type UpdateUserRequest struct {
	Status *int8 `json:"status" binding:"omitempty,oneof=0 1"`
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
	ID        uint64    `json:"id"`
	Username  string    `json:"username"`
	Status    int8      `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func ToUserResponse(user domain.User) UserResponse {
	return UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		Status:    user.Status,
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
