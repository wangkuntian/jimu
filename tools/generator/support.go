package generator

import (
	"io/fs"

	"jimu/tools/generator/support"
)

// CopyTree copies a source tree while preserving the generator's fail-closed
// symlink and filter semantics.
func CopyTree(src, dst string, filter func(rel string, d fs.DirEntry) bool) (int, []string, error) {
	return support.CopyTree(src, dst, filter)
}

// RewriteModule applies the controlled module/path rewrite used by the legacy
// framework rendering helpers.
func RewriteModule(root, from, to string) ([]string, error) {
	return support.RewriteModule(root, from, to)
}

// Tidy runs go mod tidy with the generator's isolated workspace settings.
func Tidy(dir string) error {
	return support.Tidy(dir)
}

// SelfCheck runs the generated project's build and capability gate checks.
func SelfCheck(dir string) error {
	return support.SelfCheck(dir)
}

func modulePathCollides(to, name string) bool { return support.ModulePathCollides(to, name) }

func artifactName(module string) string { return support.ArtifactName(module) }

func isRewriteTarget(base string) bool { return support.IsRewriteTarget(base) }

func relPath(root, path string) string { return support.RelPath(root, path) }
