package tenant

import (
	"errors"
	"fmt"

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
	if len(p.Roles) == 0 {
		return errors.New("tenant.provisioning.enabled requires at least one role in tenant.provisioning.roles")
	}
	names := make(map[string]bool, len(p.Roles))
	for _, role := range p.Roles {
		if role.Name == "" {
			return errors.New("tenant.provisioning.roles[].name is required")
		}
		if names[role.Name] {
			return fmt.Errorf("duplicate tenant.provisioning.roles[].name: %q", role.Name)
		}
		names[role.Name] = true
		for _, permission := range role.Permissions {
			if permission.Resource == "" || permission.Action == "" {
				return fmt.Errorf("tenant.provisioning.roles[%q].permissions entries require resource and action", role.Name)
			}
		}
	}
	if p.OwnerRole != "" && !names[p.OwnerRole] {
		return fmt.Errorf("tenant.provisioning.owner_role %q not found in tenant.provisioning.roles", p.OwnerRole)
	}
	return nil
}

var _ config.SectionConfig = (*Config)(nil)
