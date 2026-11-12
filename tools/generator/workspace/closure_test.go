package workspace

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func writeClosureFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		"go.mod": "module example.com/app\n\ngo 1.26\n",
		"cmd/server/main.go": `package main

import "example.com/app/internal/common"

func main() { common.Use() }
`,
		"cmd/server/linux.go": `//go:build linux

package main

import _ "example.com/app/internal/linuxonly"
`,
		"cmd/server/darwin.go": `//go:build darwin

package main

import _ "example.com/app/internal/darwinonly"
`,
		"cmd/server/main_test.go": `package main

import _ "example.com/app/internal/testonly"
`,
		"internal/common/common.go": `package common

func Use() {}
`,
		"internal/linuxonly/linux.go": `package linuxonly
`,
		"internal/darwinonly/darwin.go": `package darwinonly
`,
		"internal/unreachable/unreachable.go": `package unreachable
`,
		"internal/testonly/testonly.go": `package testonly
`,
	}
	for rel, content := range files {
		path := filepath.Join(dir, filepath.FromSlash(rel))
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	}
	return dir
}

func TestComputePackageClosureKeepsBuildTaggedPlatforms(t *testing.T) {
	dir := writeClosureFixture(t)
	got, err := ComputePackageClosure(context.Background(), dir, ClosureOptions{
		Roots:  []string{"example.com/app/cmd/server"},
		Module: "example.com/app",
		Platforms: []Platform{
			{GOOS: "linux", GOARCH: "amd64"},
			{GOOS: "darwin", GOARCH: "arm64"},
		},
	})
	require.NoError(t, err)
	require.True(t, got.Runtime["example.com/app/internal/linuxonly"])
	require.True(t, got.Runtime["example.com/app/internal/darwinonly"])
	require.True(t, got.Runtime["example.com/app/internal/common"])
	require.False(t, got.Runtime["example.com/app/internal/unreachable"])
}

func TestComputePackageClosureSeparatesTestOnlyPackages(t *testing.T) {
	dir := writeClosureFixture(t)
	got, err := ComputePackageClosure(context.Background(), dir, ClosureOptions{
		Roots:     []string{"example.com/app/cmd/server"},
		Module:    "example.com/app",
		Platforms: []Platform{{GOOS: "linux", GOARCH: "amd64"}},
	})
	require.NoError(t, err)
	require.True(t, got.Tests["example.com/app/internal/testonly"])
	require.False(t, got.Runtime["example.com/app/internal/testonly"])
}
