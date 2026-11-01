package app

import (
	"testing"

	"github.com/stretchr/testify/require"

	"jimu/internal/config"
)

func TestHTTPSOnlyContainerSkipsDatabaseAndRedis(t *testing.T) {
	cfg := &config.Config{
		HTTP: config.HTTPConfig{
			Host: "127.0.0.1", Port: 18080, Mode: config.HTTPModeTest,
			ReadHeaderTimeoutSec: 5, ReadTimeoutSec: 15, WriteTimeoutSec: 30,
			IdleTimeoutSec: 60, ShutdownTimeoutSec: 15, MaxBodyBytes: 1 << 20,
		},
		Management: config.ManagementConfig{Host: "127.0.0.1", Port: 19090, ProbeTimeoutSec: 1},
		Log:        config.LogConfig{Level: config.LogLevelInfo, Format: config.LogFormatConsole, Output: "stdout"},
	}

	container, err := NewContainer(cfg, nil, &CapabilityConfigs{}, nil, map[string]bool{})
	require.NoError(t, err)
	require.Nil(t, container.DB)
	require.Nil(t, container.Redis)

	application, err := Bootstrap(container, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, application)
	_ = container.Stop(t.Context())
}
