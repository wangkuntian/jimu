package tenant

import (
	"reflect"
	"strings"
	"testing"

	"jimu/internal/config"
	"jimu/internal/contract"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDescriptorDeclaresProvisioningConfiguration(t *testing.T) {
	require.Len(t, Descriptor.Configs, 1)
	assert.Equal(t, ConfigKey, Descriptor.Configs[0].Section)
	_, ok := Descriptor.Configs[0].New().(config.SectionConfig)
	assert.True(t, ok)
}

func TestProvisioningConfigurationValidation(t *testing.T) {
	assert.NoError(t, (&Config{}).Validate())
	assert.NoError(t, (&Config{Provisioning: ProvisioningConfig{
		OwnerRole: "missing",
		Roles:     []ProvisionRoleTemplate{{}},
	}}).Validate(), "disabled provisioning config skips its validation")
	valid := ProvisioningConfig{Enabled: true, OwnerRole: "owner", Roles: []ProvisionRoleTemplate{{
		Name: "owner", Permissions: []ProvisionPermission{{Resource: "/api/v1/users", Action: "GET"}},
	}}}
	assert.NoError(t, (&Config{Provisioning: valid}).Validate())

	for _, tc := range []struct {
		name string
		cfg  ProvisioningConfig
		want string
	}{
		{name: "missing roles", cfg: ProvisioningConfig{Enabled: true}, want: "tenant.provisioning.roles"},
		{name: "empty role name", cfg: ProvisioningConfig{Enabled: true, Roles: []ProvisionRoleTemplate{{}}}, want: "roles[].name"},
		{name: "duplicate role", cfg: ProvisioningConfig{Enabled: true, Roles: []ProvisionRoleTemplate{{Name: "owner"}, {Name: "owner"}}}, want: "duplicate"},
		{name: "unknown owner role", cfg: ProvisioningConfig{Enabled: true, OwnerRole: "missing", Roles: []ProvisionRoleTemplate{{Name: "owner"}}}, want: "owner_role"},
		{name: "incomplete permission", cfg: ProvisioningConfig{Enabled: true, Roles: []ProvisionRoleTemplate{{Name: "owner", Permissions: []ProvisionPermission{{Resource: "/api/v1/users"}}}}}, want: "resource and action"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := (&Config{Provisioning: tc.cfg}).Validate()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.want)
		})
	}
}

func TestConfigDecodesTenantProvisioningSection(t *testing.T) {
	v := viper.New()
	v.SetConfigType("yaml")
	require.NoError(t, v.ReadConfig(strings.NewReader("tenant:\n  provisioning:\n    enabled: true\n    owner_role: owner\n    roles:\n      - name: owner\n        permissions:\n          - resource: /api/v1/users\n            action: GET\n")))
	var cfg Config
	require.NoError(t, v.UnmarshalKey(ConfigKey, &cfg))
	assert.True(t, cfg.Provisioning.Enabled)
	assert.Equal(t, "owner", cfg.Provisioning.OwnerRole)
	require.Len(t, cfg.Provisioning.Roles, 1)
	assert.Equal(t, "/api/v1/users", cfg.Provisioning.Roles[0].Permissions[0].Resource)
}

func TestProvisionerFactoryReportsEnabledState(t *testing.T) {
	enabled := New(nil, ProvisioningConfig{Enabled: true})
	disabled := New(nil, ProvisioningConfig{})

	var enabledFactory contract.TenantProvisionerFactory = enabled
	var disabledFactory contract.TenantProvisionerFactory = disabled
	assert.True(t, enabledFactory.Enabled())
	assert.False(t, disabledFactory.Enabled())
}

func TestProvisioningConfigurationIsNotPartOfAuthContract(t *testing.T) {
	typ := reflect.TypeOf(contract.AuthConfig{})
	_, ok := typ.FieldByName("Provisioning")
	assert.False(t, ok)
}
