// Package plan turns a validated manifest into a deterministic execution plan.
// It deliberately has no dependency on Jimu framework packages.
package plan

import "jimu/tools/generator/manifest"

type Plan struct {
	Selection      manifest.Selection
	Copy           []manifest.CopyAction
	Templates      []manifest.TemplateAction
	Merges         []manifest.MergeAction
	Rewrites       []manifest.RewriteAction
	Assets         []manifest.AssetAction
	Prune          []manifest.PruneRule
	Report         manifest.ReportSpec
	GeneratedFiles []string
}
