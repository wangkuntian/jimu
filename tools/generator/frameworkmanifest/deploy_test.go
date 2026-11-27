package frameworkmanifest

import (
	"testing"

	"jimu/tools/generator/manifest"

	"github.com/stretchr/testify/require"
)

func TestDefaultDeploy(t *testing.T) {
	require.Empty(t, DefaultDeploy("minimal"))
	require.Equal(t, []string{manifest.DeployCompose}, DefaultDeploy("auth"))
	require.Equal(t, []string{
		manifest.DeployCompose,
		manifest.DeployHelm,
		manifest.DeployK8s,
		manifest.DeployObservability,
	}, DefaultDeploy("full"))
	require.Equal(t, DefaultDeploy("full"), DefaultDeploy(""))
}

func TestParseDeploy(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want []string
	}{
		{"none", "none", []string{}},
		{"all", "all", []string{
			manifest.DeployCompose,
			manifest.DeployHelm,
			manifest.DeployK8s,
			manifest.DeployObservability,
		}},
		{"duplicates", "k8s,compose,k8s", []string{manifest.DeployCompose, manifest.DeployK8s}},
		{"whitespace", " compose, observability ", []string{manifest.DeployCompose, manifest.DeployObservability}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDeploy(tt.raw)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestParseDeployRejectsInvalidInput(t *testing.T) {
	for _, raw := range []string{"unknown", "none,compose", "all,k8s", "compose,,k8s"} {
		t.Run(raw, func(t *testing.T) {
			_, err := ParseDeploy(raw)
			require.Error(t, err)
		})
	}
}

func TestHasDeploy(t *testing.T) {
	require.True(t, HasDeploy([]string{manifest.DeployCompose}, manifest.DeployCompose))
	require.False(t, HasDeploy([]string{manifest.DeployK8s}, manifest.DeployCompose))
}
