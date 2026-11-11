package app

import (
	"testing"

	"jimu/internal/contract"

	"github.com/stretchr/testify/require"
)

func TestRequiredRuntimeServicesIncludesSelectedDriver(t *testing.T) {
	caps := []contract.Descriptor{{
		Name:            "auth",
		RuntimeServices: []contract.RuntimeService{contract.RuntimeServiceMySQL},
		Drivers:         []string{"redis", "memory"},
		DriverSpecs: []contract.DriverSpec{
			{Name: "redis", RuntimeServices: []contract.RuntimeService{contract.RuntimeServiceRedis}},
			{Name: "memory"},
		},
	}}

	withRedis := requiredRuntimeServices(caps, map[string][]string{"auth": {"redis"}})
	require.True(t, withRedis[contract.RuntimeServiceMySQL])
	require.True(t, withRedis[contract.RuntimeServiceRedis])

	withMemory := requiredRuntimeServices(caps, map[string][]string{"auth": {"memory"}})
	require.True(t, withMemory[contract.RuntimeServiceMySQL])
	require.False(t, withMemory[contract.RuntimeServiceRedis])
}
