package frameworkmanifest

import (
	"testing"

	"jimu/tools/generator/manifest"

	"github.com/stretchr/testify/require"
)

func TestResolveGenerationMinimalDefaults(t *testing.T) {
	got, err := ResolveGeneration("minimal", "", nil, GenerationRequest{})
	require.NoError(t, err)
	require.Equal(t, manifest.GenerationTestsSelected, got.Tests)
	require.Equal(t, manifest.GenerationToolsAuto, got.ToolsMode)
	require.Equal(t, []string{"logcheck"}, got.ToolsResolved)
	require.Empty(t, got.Deploy)
	require.Equal(t, manifest.GenerationCLIMinimal, got.CLI)
	require.Equal(t, manifest.GenerationCLIUIDirect, got.CLIUI)
	require.Equal(t, manifest.GenerationRuntimeLean, got.Runtime)
}

func TestResolveGenerationRuntimeDefaults(t *testing.T) {
	for _, test := range []struct {
		name     string
		profile  string
		with     string
		declared []string
		want     string
	}{
		{name: "minimal", profile: "minimal", want: manifest.GenerationRuntimeLean},
		{name: "auth", profile: "auth", declared: []string{"user"}, want: manifest.GenerationRuntimeStandard},
		{name: "full", profile: "full", declared: []string{"user"}, want: manifest.GenerationRuntimeStandard},
		{name: "with", with: "user", declared: []string{"user"}, want: manifest.GenerationRuntimeStandard},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := ResolveGeneration(test.profile, test.with, test.declared, GenerationRequest{})
			require.NoError(t, err)
			require.Equal(t, test.want, got.Runtime)
		})
	}
}

func TestResolveGenerationRuntimeOverrides(t *testing.T) {
	got, err := ResolveGeneration("minimal", "", nil, GenerationRequest{Runtime: manifest.GenerationRuntimeStandard})
	require.NoError(t, err)
	require.Equal(t, manifest.GenerationRuntimeStandard, got.Runtime)

	got, err = ResolveGeneration("minimal", "", nil, GenerationRequest{
		Deploy: []string{manifest.DeployCompose, manifest.DeployObservability},
	})
	require.NoError(t, err)
	require.Equal(t, manifest.GenerationRuntimeStandard, got.Runtime,
		"minimal + observability 必须自动提升为 standard，否则 lean 不编 OTel")

	_, err = ResolveGeneration("minimal", "", []string{"user"}, GenerationRequest{Runtime: manifest.GenerationRuntimeLean})
	require.ErrorContains(t, err, "requires a project without capabilities")

	_, err = ResolveGeneration("minimal", "", nil, GenerationRequest{
		Runtime: manifest.GenerationRuntimeLean,
		Deploy:  []string{manifest.DeployObservability},
	})
	require.ErrorContains(t, err, "observability")
}

func TestResolveGenerationProfileCLIUIDefaults(t *testing.T) {
	for _, test := range []struct {
		profile string
		with    string
		want    string
	}{
		{profile: "minimal", want: manifest.GenerationCLIUIDirect},
		{profile: "auth", want: manifest.GenerationCLIUIBoth},
		{profile: "full", want: manifest.GenerationCLIUIBoth},
		{with: "user", want: manifest.GenerationCLIUIBoth},
	} {
		got, err := ResolveGeneration(test.profile, test.with, nil, GenerationRequest{})
		require.NoError(t, err)
		require.Equal(t, test.want, got.CLIUI)
	}
}

func TestResolveGenerationToolsAutoWithCapability(t *testing.T) {
	got, err := ResolveGeneration("auth", "", []string{"user"}, GenerationRequest{Tools: manifest.GenerationToolsAuto})
	require.NoError(t, err)
	require.Equal(t, []string{"checkcapabilities", "logcheck"}, got.ToolsResolved)
}

func TestResolveGenerationExplicitModesWin(t *testing.T) {
	got, err := ResolveGeneration("minimal", "", nil, GenerationRequest{
		Tests: manifest.GenerationTestsNone,
		Tools: manifest.GenerationToolsNone,
		Deploy: []string{
			manifest.DeployK8s,
			manifest.DeployCompose,
		},
		CLI:   manifest.GenerationCLIFull,
		CLIUI: manifest.GenerationCLIUIBoth,
	})
	require.NoError(t, err)
	require.Equal(t, manifest.GenerationTestsNone, got.Tests)
	require.Equal(t, manifest.GenerationToolsNone, got.ToolsMode)
	require.Empty(t, got.ToolsResolved)
	require.Equal(t, []string{manifest.DeployCompose, manifest.DeployK8s}, got.Deploy)
	require.Equal(t, manifest.GenerationCLIFull, got.CLI)
	require.Equal(t, manifest.GenerationCLIUIBoth, got.CLIUI)
}

func TestResolveGenerationRejectsUnknownValues(t *testing.T) {
	_, err := ResolveGeneration("minimal", "", nil, GenerationRequest{Tests: "sometimes"})
	require.ErrorContains(t, err, "tests")
}
