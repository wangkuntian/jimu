package workspace

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

type Platform struct {
	GOOS   string
	GOARCH string
}

type ClosureOptions struct {
	Roots     []string
	Module    string
	Platforms []Platform
}

type PackageClosure struct {
	Runtime map[string]bool
	Tests   map[string]bool
}

// ComputePackageClosure calculates the union of module packages reachable
// from Roots on every supported platform. Tests contains test-only packages
// and never expands Runtime.
func ComputePackageClosure(ctx context.Context, dir string, opts ClosureOptions) (PackageClosure, error) {
	if len(opts.Roots) == 0 {
		return PackageClosure{}, fmt.Errorf("closure roots are empty")
	}
	if opts.Module == "" {
		return PackageClosure{}, fmt.Errorf("closure module is empty")
	}
	platforms := opts.Platforms
	if len(platforms) == 0 {
		platforms = []Platform{{GOOS: "linux", GOARCH: "amd64"}}
	}
	closure := PackageClosure{Runtime: map[string]bool{}, Tests: map[string]bool{}}
	for _, platform := range platforms {
		runtimePackages, err := listPackages(ctx, dir, opts, platform, false)
		if err != nil {
			return PackageClosure{}, err
		}
		for name := range runtimePackages {
			closure.Runtime[name] = true
		}
		testPackages, err := listPackages(ctx, dir, opts, platform, true)
		if err != nil {
			return PackageClosure{}, err
		}
		for name := range testPackages {
			if !closure.Runtime[name] {
				closure.Tests[name] = true
			}
		}
	}
	return closure, nil
}

func listPackages(ctx context.Context, dir string, opts ClosureOptions, platform Platform, includeTests bool) (map[string]bool, error) {
	args := []string{"list", "-deps", "-json"}
	if includeTests {
		args = append(args, "-test")
	}
	args = append(args, opts.Roots...)
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GOWORK=off",
		"GOOS="+platform.GOOS,
		"GOARCH="+platform.GOARCH,
		"CGO_ENABLED=0",
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("go list %s/%s: %w\n%s", platform.GOOS, platform.GOARCH, err, strings.TrimSpace(stderr.String()))
	}
	decoder := json.NewDecoder(&stdout)
	packages := map[string]bool{}
	for {
		var pkg struct {
			ImportPath string
		}
		if err := decoder.Decode(&pkg); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("decode go list %s/%s: %w", platform.GOOS, platform.GOARCH, err)
		}
		if pkg.ImportPath != "" && belongsToModule(pkg.ImportPath, opts.Module) {
			packages[pkg.ImportPath] = true
		}
	}
	return packages, nil
}

func belongsToModule(importPath, module string) bool {
	return importPath == module || strings.HasPrefix(importPath, module+"/")
}
