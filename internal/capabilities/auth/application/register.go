package application

import (
	"time"

	"jimu/internal/contract"
)

// RegisterUserView 注册响应里的用户视图（不含密码）。
// 两种注册模式共用同一形状，前端只需一个类型。
type RegisterUserView struct {
	ID        uint64    `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Phone     string    `json:"phone"`
	Status    int8      `json:"status"`
	TenantID  uint64    `json:"tenant_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// RegisterTenantView 开通式注册返回的新租户视图。
// 不直接复用 tenant 能力的类型：能力之间不互相 import 内部包。
type RegisterTenantView struct {
	ID     uint64 `json:"id"`
	Code   string `json:"code"`
	Name   string `json:"name"`
	Status int8   `json:"status"`
}

// RegisterResponse 注册成功后的响应：普通注册只有 user；
// 开通式注册额外返回 tenant（注册者成为该租户 owner）。
type RegisterResponse struct {
	User   RegisterUserView    `json:"user"`
	Tenant *RegisterTenantView `json:"tenant,omitempty"`
}

// NewRegisterUserView 由账号构造注册响应视图（普通注册路径）。
func NewRegisterUserView(account contract.Account) RegisterUserView {
	return RegisterUserView{
		ID:        account.ID,
		Username:  account.Username,
		Email:     account.Email,
		Phone:     account.Phone,
		Status:    account.Status,
		TenantID:  account.TenantID,
		CreatedAt: account.CreatedAt,
		UpdatedAt: account.UpdatedAt,
	}
}

// NewRegisterResponse 由开通结果构造注册响应（开通式注册路径，含 tenant）。
func NewRegisterResponse(result *contract.ProvisionResult) RegisterResponse {
	return RegisterResponse{
		User: RegisterUserView{
			ID:        result.User.ID,
			Username:  result.User.Username,
			Email:     result.User.Email,
			Phone:     result.User.Phone,
			Status:    result.User.Status,
			TenantID:  result.User.TenantID,
			CreatedAt: result.User.CreatedAt,
			UpdatedAt: result.User.UpdatedAt,
		},
		Tenant: &RegisterTenantView{
			ID:     result.Tenant.ID,
			Code:   result.Tenant.Code,
			Name:   result.Tenant.Name,
			Status: result.Tenant.Status,
		},
	}
}
