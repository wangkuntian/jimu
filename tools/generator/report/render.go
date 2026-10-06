package report

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"jimu/tools/internal/projectmetrics"
)

const ReportPath = "docs/profiles/generated-report.md"

func Write(root string, metrics projectmetrics.Metrics, metadata Metadata, assetRoots []string) (string, error) {
	directDeps, err := projectmetrics.DirectDeps(root, metadata.Module)
	if err != nil {
		return "", fmt.Errorf("count direct dependencies: %w", err)
	}
	generated, assets, err := TreeCounts(root, assetRoots)
	if err != nil {
		return "", err
	}
	metadata.GeneratedFiles = generated
	metadata.AssetFiles = assets
	content := Render(metrics, metadata, directDeps)
	target := filepath.Join(root, filepath.FromSlash(ReportPath))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
		return "", err
	}
	return content, nil
}

func TreeCounts(root string, assetRoots []string) (generated, assets int, err error) {
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if entry.Name() == ".git" {
				return fs.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == ".jimu/manifest.json" || rel == ReportPath {
			return nil
		}
		generated++
		for _, assetRoot := range assetRoots {
			assetRoot = strings.TrimSuffix(filepath.ToSlash(assetRoot), "/")
			if rel == assetRoot || strings.HasPrefix(rel, assetRoot+"/") {
				assets++
				break
			}
		}
		return nil
	})
	return generated, assets, err
}

func Render(metrics projectmetrics.Metrics, metadata Metadata, directDeps int) string {
	capabilities := slices.Clone(metadata.Capabilities)
	if len(capabilities) == 0 {
		capabilities = slices.Clone(metrics.Capabilities)
	}
	var b strings.Builder
	b.WriteString("# 生成项目编译面报告\n\n")
	b.WriteString("> 由 `jimu new --report`（`tools/generator`）生成，**请勿手工编辑**：能力集变化后重新生成并提交本报告。\n")
	b.WriteString("> 本项目只有一个形态，形态在生成期固定；下表只记录本项目自身的实测编译面。\n\n")
	b.WriteString("## 项目概况\n\n")
	b.WriteString("| 项 | 值 |\n|---|---|\n")
	fmt.Fprintf(&b, "| 模块 | `%s` |\n", metadata.Module)
	fmt.Fprintf(&b, "| 形态 | `%s` |\n", metadata.Shape)
	fmt.Fprintf(&b, "| 声明能力（装配集） | %s |\n", joinCell(capabilities))
	fmt.Fprintf(&b, "| 迁移携带能力（schema 依赖） | %s |\n", joinCell(metadata.MigrationOnly))
	fmt.Fprintf(&b, "| 只带 domain 的能力（内核编译期依赖） | %s |\n", joinCell(metadata.DomainOnly))
	fmt.Fprintf(&b, "| 选中驱动 | %s |\n", joinCell(metadata.Drivers))
	fmt.Fprintf(&b, "| 复制的资产根 | %s |\n", joinCell(metadata.Assets))
	b.WriteString("\n## 指标口径\n\n")
	b.WriteString("| 指标 | 口径 |\n|---|---|\n")
	b.WriteString("| 生成文件数 | 生成树里的全部文件数（不含 manifest 与本报告） |\n")
	b.WriteString("| 本模块 Go 文件 / 代码行（闭包） | `go/packages` 载入 `./cmd/server` 的 import 闭包，只统计本模块的非 `_test.go` 文件 |\n")
	b.WriteString("| go.mod 直接依赖 | `go list -m` 的非空非间接依赖数（不含主模块自身） |\n")
	b.WriteString("| 迁移数 / 表数 / 路由数 | 由 manifest 导出的静态 ReportSpec 提供 |\n")
	b.WriteString("| 重型依赖 | 闭包命中 `tools/internal/heavydeps` 的展示名，`-` 表示零 |\n")
	b.WriteString("| 资产文件数 | 复制进来的资产根下的文件数 |\n\n")
	b.WriteString("## 编译面\n\n")
	b.WriteString("| 指标 | 值 |\n|---|---:|\n")
	fmt.Fprintf(&b, "| 生成文件数 | %d |\n", metadata.GeneratedFiles)
	fmt.Fprintf(&b, "| 本模块 Go 文件（闭包） | %d |\n", metrics.Files)
	fmt.Fprintf(&b, "| 本模块代码行（闭包） | %d |\n", metrics.Lines)
	fmt.Fprintf(&b, "| go.mod 直接依赖 | %d |\n", directDeps)
	fmt.Fprintf(&b, "| 迁移数 | %d |\n", metrics.Migrations)
	fmt.Fprintf(&b, "| 表数 | %d |\n", metrics.Tables)
	fmt.Fprintf(&b, "| 路由数 | %d |\n", metrics.Routes)
	fmt.Fprintf(&b, "| 重型依赖 | %s |\n", joinCell(metrics.HeavyDeps))
	fmt.Fprintf(&b, "| 资产文件数 | %d |\n", metadata.AssetFiles)
	b.WriteString("\n## 解析集（`jimu migrate` / `seed` 的能力清单）\n\n")
	fmt.Fprintf(&b, "| 装配的能力（按装配顺序） |\n|---|\n| %s |\n", joinCell(capabilities))
	return b.String()
}

func joinCell(items []string) string {
	if len(items) == 0 {
		return "-"
	}
	return strings.Join(items, ", ")
}
