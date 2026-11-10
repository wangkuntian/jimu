package generator

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// 内核解耦的钉子：internal/app 不得再 import 任何能力 domain 包。
// seed 所需模型已下沉到 internal/kernel/seedmodel（见 Task 1）。
func TestAppDoesNotImportCapabilityDomains(t *testing.T) {
	deps := goListDeps(t, "./internal/app")
	for _, dep := range deps {
		require.NotContains(t, dep, "jimu/internal/capabilities",
			"internal/app must not depend on capability packages")
	}
}

func TestMinimalPresetDropsUnreachableKernelPackages(t *testing.T) {
	dir := generateForTest(t, NewOptions{Template: "minimal"})
	require.NoDirExists(t, filepath.Join(dir, "internal/e2e"),
		"internal/e2e only contains tests and is unreachable with Tests=false")
	require.NoDirExists(t, filepath.Join(dir, "internal/capabilities", "user"),
		"a 0-capability project must not ship capability packages")
	for _, kept := range []string{
		"internal/kernel/http",
		"internal/kernel/metrics",
		"internal/assembly",
		"cmd/server",
		"cmd/cli",
	} {
		require.DirExists(t, filepath.Join(dir, kept), "%s is required by the entrypoint", kept)
	}
}

func TestMinimalPresetDropsOnlyUnreachableKernelSubpackages(t *testing.T) {
	dir := generateForTest(t, NewOptions{Template: "minimal"})
	require.NoDirExists(t, filepath.Join(dir, "internal/kernel", "cache"),
		"internal/kernel/cache is only reachable through capabilities")
	require.DirExists(t, filepath.Join(dir, "internal/kernel", "http"))
	require.DirExists(t, filepath.Join(dir, "internal/kernel", "db"))
}

func TestFullPresetKeepsEveryKernelPackage(t *testing.T) {
	dir := generateForTest(t, NewOptions{Template: "full"})
	for _, kept := range []string{"internal/kernel/http", "internal/kernel/db", "internal/shared"} {
		require.DirExists(t, filepath.Join(dir, kept))
	}
}

// goListDeps 返回某个包（含子包展开）的依赖 import path 列表。
func goListDeps(t *testing.T, pkg string) []string {
	t.Helper()
	root := frameworkRootForTest(t)
	cmd := exec.Command("go", "list", "-deps", pkg)
	cmd.Dir = root
	out, err := cmd.Output()
	require.NoError(t, err, "go list -deps %s in %s: %s", pkg, root, strings.TrimSpace(string(out)))
	return strings.Fields(string(out))
}
