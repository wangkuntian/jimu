package application

import (
	"encoding/json"
	"testing"
	"time"

	"jimu/internal/contract"

	"github.com/stretchr/testify/require"
)

// TestRegisterResponseOmitsTenantForNormalRegistration 普通注册只序列化 user：
// tenant 为 nil 且带 omitempty，响应形状与开通式注册统一为 {user, tenant?}。
func TestRegisterResponseOmitsTenantForNormalRegistration(t *testing.T) {
	created := time.Date(2026, 10, 13, 9, 30, 0, 0, time.UTC)
	resp := RegisterResponse{User: NewRegisterUserView(contract.Account{
		ID:        7,
		Username:  "acme",
		Password:  "hashed-secret",
		Email:     "acme@example.com",
		Phone:     "13800000000",
		Status:    1,
		TenantID:  1,
		CreatedAt: created,
		UpdatedAt: created,
	})}

	raw, err := json.Marshal(resp)
	require.NoError(t, err)
	require.NotContains(t, string(raw), `"tenant":`, "普通注册不得出现 tenant 字段")
	require.NotContains(t, string(raw), "password", "注册响应不得泄漏密码")

	require.JSONEq(t, `{"user":{"id":7,"username":"acme","email":"acme@example.com",`+
		`"phone":"13800000000","status":1,"tenant_id":1,`+
		`"created_at":"2026-10-13T09:30:00Z","updated_at":"2026-10-13T09:30:00Z"}}`, string(raw))
}

// TestRegisterResponseCarriesTenantForProvisionedRegistration 开通式注册返回 {user, tenant}，
// tenant 是注册者成为 owner 的新租户。
func TestRegisterResponseCarriesTenantForProvisionedRegistration(t *testing.T) {
	resp := NewRegisterResponse(&contract.ProvisionResult{
		User:   contract.ProvisionedUser{ID: 7, Username: "acme", Status: 1, TenantID: 42},
		Tenant: contract.ProvisionedTenant{ID: 42, Code: "acme", Name: "Acme", Status: 1},
	})

	raw, err := json.Marshal(resp)
	require.NoError(t, err)
	require.Contains(t, string(raw), `"tenant":{"id":42,"code":"acme","name":"Acme","status":1}`)
	require.Contains(t, string(raw), `"user":{"id":7,"username":"acme"`)
}
