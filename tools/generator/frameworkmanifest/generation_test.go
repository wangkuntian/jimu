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
