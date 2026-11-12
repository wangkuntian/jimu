package frameworkmanifest

import (
	"fmt"
	"slices"

	"jimu/tools/generator/manifest"
)

// GenerationRequest is the raw, user-facing generation policy. Deploy is nil
// when the caller wants the profile default; a non-nil empty slice means none.
type GenerationRequest struct {
	Tests  string
	Tools  string
	Deploy []string
	CLI    string
	CLIUI  string
}

// ResolveGeneration turns raw flags into the policy persisted in a v4
// manifest. It is pure and does not inspect the framework or filesystem.
func ResolveGeneration(profile, with string, declared []string, raw GenerationRequest) (manifest.Generation, error) {
	generation := defaultGeneration(profile, with, declared)

	if raw.Tests != "" {
		if !slices.Contains([]string{manifest.GenerationTestsSelected, manifest.GenerationTestsAll, manifest.GenerationTestsNone}, raw.Tests) {
			return manifest.Generation{}, fmt.Errorf("--tests must be selected, all, or none")
		}
		generation.Tests = raw.Tests
	}
	if raw.Tools != "" {
		if !slices.Contains([]string{manifest.GenerationToolsAuto, manifest.GenerationToolsAll, manifest.GenerationToolsNone}, raw.Tools) {
			return manifest.Generation{}, fmt.Errorf("--tools must be auto, all, or none")
		}
		generation.ToolsMode = raw.Tools
	}
	generation.ToolsResolved = resolveTools(generation.ToolsMode, declared)

	if raw.Deploy != nil {
		deploy, err := normalizeDeploy(raw.Deploy)
		if err != nil {
			return manifest.Generation{}, err
		}
		generation.Deploy = deploy
	}
	if raw.CLI != "" {
		if !slices.Contains([]string{manifest.GenerationCLIMinimal, manifest.GenerationCLIFull}, raw.CLI) {
			return manifest.Generation{}, fmt.Errorf("--cli-commands must be minimal or full")
		}
		generation.CLI = raw.CLI
	}
	if raw.CLIUI != "" {
		if !slices.Contains([]string{manifest.GenerationCLIUIDirect, manifest.GenerationCLIUIBoth}, raw.CLIUI) {
			return manifest.Generation{}, fmt.Errorf("--cli must be both or direct")
		}
		generation.CLIUI = raw.CLIUI
	}
	return generation, nil
}

func defaultGeneration(profile, with string, declared []string) manifest.Generation {
	generation := manifest.Generation{
		RuntimeClosure: false,
		Tests:          manifest.GenerationTestsSelected,
		ToolsMode:      manifest.GenerationToolsAuto,
		Deploy:         []string{},
		CLI:            manifest.GenerationCLIMinimal,
		CLIUI:          manifest.GenerationCLIUIDirect,
	}
	switch profile {
	case "minimal":
		return generation
	case "auth":
		generation.ToolsMode = manifest.GenerationToolsAll
		generation.Deploy = []string{manifest.DeployHelm, manifest.DeployK8s}
		generation.CLI = manifest.GenerationCLIFull
		return generation
	case "full":
		generation.ToolsMode = manifest.GenerationToolsAll
		generation.Deploy = []string{manifest.DeployHelm, manifest.DeployK8s, manifest.DeployObservability}
		generation.CLI = manifest.GenerationCLIFull
		return generation
	}
	if profile == "" && with != "" {
		generation.Deploy = []string{manifest.DeployHelm, manifest.DeployK8s}
		if len(declared) > 0 {
			generation.CLI = manifest.GenerationCLIFull
		}
	}
	return generation
}

func resolveTools(mode string, declared []string) []string {
	switch mode {
	case manifest.GenerationToolsNone:
		return []string{}
	case manifest.GenerationToolsAll:
		return []string{"checkcapabilities", "heavydeps", "logcheck", "profileassets"}
	default:
		if len(declared) == 0 {
			return []string{"logcheck"}
		}
		return []string{"checkcapabilities", "logcheck"}
	}
}

func normalizeDeploy(values []string) ([]string, error) {
	if len(values) == 0 {
		return []string{}, nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		switch value {
		case "none":
			if len(values) != 1 {
				return nil, fmt.Errorf("--deploy=none cannot be combined with other components")
			}
			return []string{}, nil
		case "all":
			if len(values) != 1 {
				return nil, fmt.Errorf("--deploy=all cannot be combined with other components")
			}
			return []string{manifest.DeployCompose, manifest.DeployHelm, manifest.DeployK8s, manifest.DeployObservability}, nil
		case manifest.DeployCompose, manifest.DeployK8s, manifest.DeployHelm, manifest.DeployObservability:
			out = append(out, value)
		default:
			return nil, fmt.Errorf("--deploy has unknown component %q", value)
		}
	}
	slices.Sort(out)
	return slices.Compact(out), nil
}
