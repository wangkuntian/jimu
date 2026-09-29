package authmodule

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestValidateConfigProvisioningRequiresPublicRegistration auth 段的跨字段校验：
// 开通式注册必须同时开启公开注册（原 config.validateCommon 语义）。
func TestValidateConfigProvisioningRequiresPublicRegistration(t *testing.T) {
	err := validateConfig(&Config{
		Provisioning: ProvisioningConfig{Enabled: true},
	})
	require.ErrorIs(t, err, errProvisioningRequiresPublicRegistration)

	require.NoError(t, validateConfig(&Config{
		PublicRegistration: true,
		Provisioning:       ProvisioningConfig{Enabled: true},
	}))
	require.NoError(t, validateConfig(&Config{}))
}
