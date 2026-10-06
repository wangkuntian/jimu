package passkey

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"jimu/internal/config"
)

const ConfigKey = "passkey"

type Config struct {
	Enabled       bool     `mapstructure:"enabled"`
	RPDisplayName string   `mapstructure:"rp_display_name"`
	RPID          string   `mapstructure:"rp_id"`
	RPOrigins     []string `mapstructure:"rp_origins"`
	SessionTTLMin int      `mapstructure:"session_ttl_min"`
}

func (c *Config) ApplyDefaults() {}

func (c Config) Validate() error {
	if !c.Enabled {
		return nil
	}
	if strings.TrimSpace(c.RPID) == "" {
		return errors.New("passkey.rp_id is required when enabled")
	}
	if len(c.RPOrigins) == 0 {
		return errors.New("passkey.rp_origins is required when enabled")
	}
	for _, origin := range c.RPOrigins {
		u, err := url.Parse(origin)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return fmt.Errorf("passkey.rp_origins entry %q must be an absolute http(s) origin", origin)
		}
	}
	if c.SessionTTLMin < 0 {
		return errors.New("passkey.session_ttl_min must not be negative")
	}
	return nil
}

var _ config.SectionConfig = (*Config)(nil)
