package frameworkmanifest

import (
	"fmt"
	"slices"
	"strings"

	"jimu/tools/generator/manifest"
)

var deployComponents = []string{
	manifest.DeployCompose,
	manifest.DeployHelm,
	manifest.DeployK8s,
	manifest.DeployObservability,
}

// ParseDeploy parses a comma-separated deploy component list.
func ParseDeploy(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []string{}, nil
	}
	values := strings.Split(raw, ",")
	return normalizeDeploy(values)
}

// HasDeploy reports whether a deploy component is selected.
func HasDeploy(components []string, name string) bool {
	return slices.Contains(components, name)
}

// DefaultDeploy returns the deploy components implied by a generation profile.
func DefaultDeploy(profile string) []string {
	switch profile {
	case "minimal":
		return []string{}
	case "auth":
		return []string{manifest.DeployCompose}
	default:
		return []string{
			manifest.DeployCompose,
			manifest.DeployHelm,
			manifest.DeployK8s,
			manifest.DeployObservability,
		}
	}
}

func normalizeDeploy(values []string) ([]string, error) {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			return nil, fmt.Errorf("--deploy contains an empty component")
		}
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
			return slices.Clone(deployComponents), nil
		default:
			if !slices.Contains(deployComponents, value) {
				return nil, fmt.Errorf("--deploy has unknown component %q", value)
			}
			out = append(out, value)
		}
	}
	slices.Sort(out)
	return slices.Compact(out), nil
}
