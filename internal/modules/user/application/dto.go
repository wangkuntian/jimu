package application

import "time"

type CreateUserRequest struct {
	Username string `json:"username" binding:"required,min=3,max=64"`
	Password string `json:"password" binding:"required,min=6,max=32"`
}

type UpdateUserRequest struct {
	Status *int8 `json:"status" binding:"omitempty,oneof=0 1"`
}

type UserResponse struct {
	ID        uint64    `json:"id"`
	Username  string    `json:"username"`
	Status    int8      `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}
