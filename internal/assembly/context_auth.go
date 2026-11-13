package assembly

import "jimu/internal/contract"

// AuthConfig returns the validated auth section through its contract projection.
func (c *Context) AuthConfig() contract.AuthConfig {
	v, ok := c.CapabilityConfigs().Section("auth").(interface{ PortView() contract.AuthConfig })
	if !ok {
		return contract.AuthConfig{}
	}
	return v.PortView()
}
