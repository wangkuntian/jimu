package jimuassets

import (
	"io/fs"
	"os"
	"os/exec"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplateBundleMatchesTrackedFiles(t *testing.T) {
	root, err := os.Getwd()
	require.NoError(t, err)

	args := append([]string{"-C", root, "ls-files", "--cached", "--others", "--exclude-standard", "--"}, BundleRoots...)
	output, err := exec.Command("git", args...).Output()
	require.NoError(t, err)

	tracked := map[string]bool{}
	for _, path := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if path != "" && !isBundleTransient(path) {
			tracked[path] = true
		}
	}

	embedded := map[string]bool{}
	err = fs.WalkDir(TemplateBundle, ".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.IsDir() && !isBundleTransient(path) {
			embedded[path] = true
		}
		return nil
	})
	require.NoError(t, err)

	var missing, extra []string
	for path := range tracked {
		if !embedded[path] {
			missing = append(missing, path)
		}
	}
	for path := range embedded {
		if !tracked[path] {
			extra = append(extra, path)
		}
	}
	sort.Strings(missing)
	sort.Strings(extra)
	assert.Empty(t, missing, "tracked bundle files missing from embed")
	assert.Empty(t, extra, "embed contains untracked bundle files")
}

func isBundleTransient(path string) bool {
	return path == ".DS_Store" || strings.HasSuffix(path, "/.DS_Store") ||
		strings.Contains(path, "/.idea/") || strings.Contains(path, "/.vscode/") ||
		strings.HasSuffix(path, ".tmp-") || strings.Contains(path, "/.tmp-")
}
