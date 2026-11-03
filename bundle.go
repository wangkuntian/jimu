// Package jimuassets owns the versioned template bundle embedded in the release CLI.
package jimuassets

import "embed"

// BundleRoots are the tracked paths that make up the template bundle.
var BundleRoots = []string{
	"go.mod",
	"go.sum",
	".golangci.yml",
	".gitignore",
	"internal",
	"cmd/server",
	"cmd/cli",
	"conf",
	"configs",
	"deploy",
	"proto",
	"docs/openapi",
	"tools/generator/render/templates",
	"tools/internal",
	"tools/profileoverlay",
	"tools/profileassets",
	"tools/checkcapabilities",
	"tools/composereport",
	"tools/logcheck",
}

// TemplateBundle is the source set used by the release CLI to generate projects.
//
//go:embed go.mod go.sum .golangci.yml .gitignore all:internal all:cmd/server all:cmd/cli all:conf all:configs all:deploy all:proto all:docs/openapi all:tools/generator/render/templates all:tools/internal all:tools/profileoverlay all:tools/profileassets all:tools/checkcapabilities all:tools/composereport all:tools/logcheck
var TemplateBundle embed.FS

// Build metadata is injected by the release workflow through -ldflags.
var (
	BundleVersion   = "dev"
	BundleCommit    = "dev"
	BundleDirty     = "true"
	ContractVersion = "v0.1"
)
