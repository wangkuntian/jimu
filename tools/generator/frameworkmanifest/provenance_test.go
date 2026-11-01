package frameworkmanifest

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"jimu/internal/profiles"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFrameworkProvenanceMatchesEveryGitWorktree(t *testing.T) {
	root := repositoryRoot(t)
	paths := gitWorktreePaths(t, root)
	require.NotEmpty(t, paths)

	for _, path := range paths {
		if _, err := os.Stat(path); err != nil {
			continue
		}
		t.Run(filepath.Base(path), func(t *testing.T) {
			got, err := frameworkProvenance(path)
			require.NoError(t, err)
			assert.Equal(t, "jimu", got.Module)
			assert.Equal(t, profiles.Version, got.Version)
			assert.Equal(t, mustGitOutput(t, path, "rev-parse", "--verify", "HEAD^{commit}"), got.Commit)
			wantDirty := strings.TrimSpace(mustGitOutput(t, path, "status", "--porcelain=v1", "--untracked-files=normal")) != "" ||
				profiles.Version == "dev" || strings.Contains(profiles.Version, "-dev")
			assert.Equal(t, wantDirty, got.Dirty)
		})
	}
}

func TestFrameworkProvenanceSupportsLinkedWorktree(t *testing.T) {
	root := repositoryRoot(t)
	info, err := os.Stat(filepath.Join(root, ".git"))
	require.NoError(t, err)
	if info.IsDir() {
		t.Skip("当前 checkout 是普通 clone，不是 linked git worktree")
	}

	got, err := frameworkProvenance(root)
	require.NoError(t, err)
	assert.Equal(t, mustGitOutput(t, root, "rev-parse", "--verify", "HEAD^{commit}"), got.Commit)
}

func TestFrameworkProvenanceRejectsNonGitDirectory(t *testing.T) {
	_, err := frameworkProvenance(t.TempDir())
	require.ErrorContains(t, err, "rev-parse")
}

func gitWorktreePaths(t *testing.T, root string) []string {
	t.Helper()
	var paths []string
	for _, line := range strings.Split(mustGitOutput(t, root, "worktree", "list", "--porcelain"), "\n") {
		if path, ok := strings.CutPrefix(line, "worktree "); ok {
			paths = append(paths, path)
		}
	}
	return paths
}

func mustGitOutput(t *testing.T, root string, args ...string) string {
	t.Helper()
	commandArgs := append([]string{"-C", root}, args...)
	output, err := exec.Command("git", commandArgs...).CombinedOutput()
	require.NoError(t, err, "git %s:\n%s", strings.Join(commandArgs, " "), output)
	return strings.TrimSpace(string(output))
}
