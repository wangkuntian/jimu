package tenant

import (
	"jimu/internal/capabilities/tenant/application"
	"jimu/internal/config"
)

const ConfigKey = "tenant"

type Config struct {
	Provisioning application.ProvisioningConfig `mapstructure:"provisioning"`
}

type (
	ProvisioningConfig    = application.ProvisioningConfig
	ProvisionRoleTemplate = application.ProvisionRoleTemplate
	ProvisionPermission   = application.ProvisionPermission
)

func (c *Config) ApplyDefaults() {}

func (c Config) Validate() error {
	return validateProvisioning(c.Provisioning)
}

func validateProvisioning(p application.ProvisioningConfig) error {
	if !p.Enabled {
		return nil
	}
	if _, err := application.EffectiveProvisioningRoles(p); err != nil {
		return err
	}
	return nil
}

var _ config.SectionConfig = (*Config)(nil)
