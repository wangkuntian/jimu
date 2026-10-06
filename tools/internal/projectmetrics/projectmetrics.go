// Package projectmetrics 是「编译面度量」的共享原语（P2.7 从 tools/composereport 抽出）：
// 报告工具（tools/composereport）与生成器（`jimu new --report`）共用一份口径，避免两处漂移
// （先例：tools/internal/heavydeps、tools/internal/profileassets）。
//
// 全部指标不连库、不连 Redis、不启动监听：路由数在裸 *gin.Engine 上注册后统计。
package projectmetrics

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"os/exec"
	"slices"
	"strings"

	"jimu/internal/assembly"
	"jimu/internal/capabilities/catalog"
	"jimu/internal/contract"
	"jimu/tools/internal/heavydeps"

	"github.com/gin-gonic/gin"
	"golang.org/x/tools/go/packages"
)

// Metrics 是一个装配（本仓的形态、或生成项目唯一形态）的编译面实测值。
//
// Profile/BinaryBytes 只有本仓的多形态报告填（生成项目只有一个形态、也不构建二进制），
// 生成项目的报告不呈现这两项。
type Metrics struct {
	Profile      string
	BinaryBytes  int64
	Routes       int
	Migrations   int
	Tables       int
	Files        int
	Lines        int
	HeavyDeps    []string
	Capabilities []string
}

func init() {
	// 报告工具只统计路由集合，不需要 gin 的路由注册调试输出（测试同样受益）。
	gin.SetMode(gin.ReleaseMode)
}

// Of 度量装配清单 asm 在 root 里的编译面：路由 / 迁移数 / 表数 / 本模块闭包的文件数与代码行 /
// 重型依赖 / 解析集。
//
// root 必须是 asm 所在的模块树 —— 本仓多形态报告传仓库根 + 该形态的内存 overlay；生成项目
// 单形态传生成项目根 + nil overlay（提交态的 internal/profiles/active 就是该形态，生成项目
// 没有构建期叠加）。
//
// modulePath 是本模块的 import 前缀（本仓 = jimu；生成项目 = --module 的值），闭包只统计它的包。
// DirectDeps 不在这里：它是 module 级指标，本仓报告只数一次、生成项目报告也只数一次。
func Of(root, modulePath string, asm assembly.Assembly, overlay map[string][]byte) (Metrics, error) {
	res, err := assembly.ProbeAssemblyAt(root, asm, nil)
	if err != nil {
		return Metrics{}, fmt.Errorf("probe %s: %w", asm.Name, err)
	}
	// 迁移/表口径 = 迁移集（声明集 ∪ schema 依赖，与 cmd/cli 的 migrate/seed 同一来源）：
	// 只按解析集算会与 CLI 实际执行的迁移不一致（如 minimal 会一并迁移 tenant 的建表/加列）。
	migDescs := catalog.MigrationSet(namesSet(res.Capabilities))
	migrations, err := MigrationCount(migDescs)
	if err != nil {
		return Metrics{}, fmt.Errorf("count migrations of %s: %w", asm.Name, err)
	}
	files, lines, heavy, err := ClosureSize(root, modulePath, overlay)
	if err != nil {
		return Metrics{}, fmt.Errorf("measure closure of %s: %w", asm.Name, err)
	}
	return Metrics{
		Profile:      asm.Name,
		Routes:       RouteCount(res.Modules),
		Migrations:   migrations,
		Tables:       TableCount(migDescs),
		Files:        files,
		Lines:        lines,
		HeavyDeps:    heavy,
		Capabilities: res.Capabilities,
	}, nil
}

// namesSet 把能力名切片转成集合（供 catalog.MigrationSet 使用）。
func namesSet(names []string) map[string]bool {
	out := make(map[string]bool, len(names))
	for _, n := range names {
		out[n] = true
	}
	return out
}

// TableCount 表数＝各 Descriptor.Owns 的并集大小（同一张表重复声明只算一次）。
func TableCount(descs []contract.Descriptor) int {
	seen := map[string]bool{}
	for _, d := range descs {
		for _, t := range d.Owns {
			seen[t] = true
		}
	}
	return len(seen)
}

// MigrationCount 迁移数＝各 Descriptor.Migrations 里 migrations/mysql 下的 .sql 文件数。
// postgres 与 mysql 迁移同名同数（`check-capabilities` 同一口径），数一遍不重复计数。
func MigrationCount(descs []contract.Descriptor) (int, error) {
	n := 0
	for _, d := range descs {
		if d.Migrations == nil {
			continue
		}
		err := fs.WalkDir(d.Migrations, "migrations/mysql", func(path string, e fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !e.IsDir() && strings.HasSuffix(path, ".sql") {
				n++
			}
			return nil
		})
		if err != nil {
			return 0, fmt.Errorf("capability %q: %w", d.Name, err)
		}
	}
	return n, nil
}

// RouteCount 把模块注册到裸 *gin.Engine 后统计 r.Routes()。运行时组合根对声明
// MountProtected 的能力用 router.Group("", protected...) 挂载，空 relativePath 的 Group
// 不改变任何路径（只追加 handler），因此裸引擎上的路由集合与实际启动逐条一致 —— 这里量
// 的是「注册了哪些路由」，不涉及监听、中间件或鉴权语义。
func RouteCount(modules []contract.Module) int {
	r := gin.New()
	for _, m := range modules {
		m.RegisterHTTP(r)
	}
	return len(r.Routes())
}

// ClosureSize 统计 modulePath 前缀的非 _test.go 的 .go 文件数与行数，并收集闭包（含第三方包）
// 命中的重型依赖展示名（去重升序）。overlay 为 nil 时按提交态度量；非 nil 时按该 overlay 度量
// （本仓多形态的构建期叠加）。root 是 packages.Load 的工作目录、modulePath 是统计前缀。
// go/packages 在只请求名称/文件/import 图时等价于 go list -deps，不做类型检查。
func ClosureSize(root, modulePath string, overlay map[string][]byte) (files, lines int, heavy []string, err error) {
	cfg := &packages.Config{
		Mode:    packages.NeedName | packages.NeedFiles | packages.NeedImports | packages.NeedDeps,
		Dir:     root,
		Overlay: overlay,
	}
	pkgs, err := packages.Load(cfg, "./cmd/server")
	if err != nil {
		return 0, 0, nil, err
	}
	if len(pkgs) == 0 {
		return 0, 0, nil, fmt.Errorf("no packages matched ./cmd/server under %s", root)
	}

	var loadErrs []string
	seenFile := map[string]bool{}
	heavySet := map[string]bool{}
	packages.Visit(pkgs, func(p *packages.Package) bool {
		for _, e := range p.Errors {
			loadErrs = append(loadErrs, e.Error())
		}
		if dep := heavydeps.Of(p.PkgPath); dep != "" {
			heavySet[dep] = true
		}
		if !strings.HasPrefix(p.PkgPath, modulePath+"/") {
			return true
		}
		for _, f := range p.GoFiles {
			if strings.HasSuffix(f, "_test.go") || seenFile[f] {
				continue
			}
			seenFile[f] = true
			n, err := CountLines(f)
			if err != nil {
				loadErrs = append(loadErrs, err.Error())
				continue
			}
			files++
			lines += n
		}
		return true
	}, nil)
	if len(loadErrs) > 0 {
		return 0, 0, nil, fmt.Errorf("load package graph: %s", strings.Join(loadErrs, "; "))
	}
	return files, lines, slices.Sorted(maps.Keys(heavySet)), nil
}

// CountLines 计文件行数：换行符个数，末行无换行时补 1。
func CountLines(path string) (int, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	if len(b) == 0 {
		return 0, nil
	}
	n := bytes.Count(b, []byte("\n"))
	if b[len(b)-1] != '\n' {
		n++
	}
	return n, nil
}

// DirectDeps 返回 go.mod 的直接依赖数（`go list -m` 输出里排除主模块自身）。
func DirectDeps(root, modulePath string) (int, error) {
	cmd := exec.CommandContext(context.Background(), "go", "list", "-m", "-f", "{{if not .Indirect}}{{.Path}}{{end}}", "all")
	cmd.Dir = root
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("go list -m all: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	n := 0
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && line != modulePath {
			n++
		}
	}
	return n, nil
}
