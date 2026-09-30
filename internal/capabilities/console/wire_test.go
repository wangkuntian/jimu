package console

import (
	"testing"

	"jimu/internal/app"
	"jimu/internal/assembly"
	"jimu/internal/config"
	"jimu/internal/contract"

	"github.com/stretchr/testify/require"
)

func TestConsoleWireWithoutWSPort(t *testing.T) {
	a := assembly.Assembly{
		Name:         "console-test",
		Capabilities: []assembly.Capability{{Descriptor: Descriptor, Wire: Wire}},
	}
	_, modules, err := assembly.WireFor(&app.Container{Config: &config.Config{}}, nil, nil, a,
		[]contract.Descriptor{Descriptor})
	require.NoError(t, err)
	require.Len(t, modules, 1)
	require.Nil(t, modules[0].(*Module).ws)
}
