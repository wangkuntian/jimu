package authmodule

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestValidateConfigProvisioningRequiresPublicRegistration 验证 tenant 开通式注册
// 必须同时开启 auth 公开注册。
func TestValidateConfigProvisioningRequiresPublicRegistration(t *testing.T) {
	err := validateConfig(&Config{}, true)
	require.ErrorIs(t, err, errProvisioningRequiresPublicRegistration)

	require.NoError(t, validateConfig(&Config{PublicRegistration: true}, true))
	require.NoError(t, validateConfig(&Config{}, false))
}
