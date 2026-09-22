// Command composereport 生成各形态（profile）的「编译面」报告。
//
// 报告回答「层②（profile 入口包）究竟改变了什么」：二进制大小、路由数、迁移数与表数逐
// 形态实测，本仓 import 闭包的代码行数/文件数按模块内包统计；同时写明 go.mod 直接依赖
// 数在各形态间**完全相同**（Go 的依赖裁剪作用于整个 module，设计 §6.3/§11 的层②边界）。
// 生成物 docs/profiles/compose-report.md 入库，供 CI 归档对比。
//
// 全部指标不连库、不连 Redis、不启动监听：路由数在裸 *gin.Engine 上注册后统计。
package main

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"jimu/internal/assembly"
	"jimu/internal/contract"
	"jimu/internal/profiles/enterprise"
	"jimu/internal/profiles/full"
	"jimu/internal/profiles/machine"
	"jimu/internal/profiles/minimal"
	"jimu/internal/profiles/saas"

	"github.com/gin-gonic/gin"
	"golang.org/x/tools/go/packages"
)

// outputPath 报告生成位置（相对仓库根）。
const outputPath = "docs/profiles/compose-report.md"

// modulePath 本模块的 import 前缀：代码量只统计本模块的包（形态差异全部来自本仓代码，
// 第三方依赖的代码量会把信号淹没，那部分由二进制大小衡量）。
const modulePath = "jimu"

// profileNames 形态的固定顺序：报告行序与断言取值都依赖它，避免 map 迭代顺序。
var profileNames = []string{"full", "minimal", "saas", "enterprise", "machine"}

// Metrics 是一个形态的编译面实测值。
type Metrics struct {
	Profile      string
	BinaryBytes  int64
	Routes       int
	Migrations   int
	Tables       int
	Files        int
	Lines        int
	Capabilities []string
}

func init() {
	// 报告工具只统计路由集合，不需要 gin 的路由注册调试输出（测试同样受益）。
	gin.SetMode(gin.ReleaseMode)
}

func main() {
	root, err := os.Getwd()
	if err != nil {
		fail(err)
	}
	ms, err := measureAll(root)
	if err != nil {
		fail(err)
	}
	deps, err := directDeps(root)
	if err != nil {
		fail(err)
	}
	out := filepath.Join(root, outputPath)
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		fail(err)
	}
	if err := os.WriteFile(out, []byte(renderReport(ms, deps)), 0o644); err != nil {
		fail(err)
	}
	fmt.Printf("✅ compose-report: %s\n", outputPath)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "❌ compose-report:", err)
	os.Exit(1)
}

// measureAll 按 profileNames 顺序实测全部形态。
//
// 五个二进制并行构建（互不共享状态），随后逐形态顺序探测：ProbeAssembly 会临时切换
// 工作目录以吸收构造期相对路径副作用，因此不能并发。
func measureAll(root string) ([]Metrics, error) {
	asms := assemblies()

	sizes := make([]int64, len(profileNames))
	errs := make([]error, len(profileNames))
	var wg sync.WaitGroup
	for i, name := range profileNames {
		wg.Add(1)
		go func(i int, name string) {
			defer wg.Done()
			sizes[i], errs[i] = buildSize(root, name)
		}(i, name)
	}
	wg.Wait()

	out := make([]Metrics, 0, len(profileNames))
	for i, name := range profileNames {
		if errs[i] != nil {
			return nil, errs[i]
		}
		a := asms[name]
		res, err := assembly.ProbeAssembly(a, nil)
		if err != nil {
			return nil, fmt.Errorf("probe %s: %w", name, err)
		}
		descs := resolvedDescriptors(a, res.Capabilities)
		migrations, err := migrationCount(descs)
		if err != nil {
			return nil, fmt.Errorf("count migrations of %s: %w", name, err)
		}
		files, lines, err := closureSize(root, name)
		if err != nil {
			return nil, fmt.Errorf("measure closure of %s: %w", name, err)
		}
		out = append(out, Metrics{
			Profile:      name,
			BinaryBytes:  sizes[i],
			Routes:       routeCount(res.Modules),
			Migrations:   migrations,
			Tables:       tableCount(descs),
			Files:        files,
			Lines:        lines,
			Capabilities: res.Capabilities,
		})
	}
	return out, nil
}

// assemblies 各形态的能力清单。
func assemblies() map[string]assembly.Assembly {
	return map[string]assembly.Assembly{
		"full":       full.Assembly(),
		"minimal":    minimal.Assembly(),
		"saas":       saas.Assembly(),
		"enterprise": enterprise.Assembly(),
		"machine":    machine.Assembly(),
	}
}

// resolvedDescriptors 取形态清单中真正进入解析集的 Descriptor，顺序同装配顺序。
// ProbeResult 只给出能力名，表/迁移口径需要 Descriptor 本身（Owns/Migrations）。
func resolvedDescriptors(a assembly.Assembly, names []string) []contract.Descriptor {
	want := make(map[string]bool, len(names))
	for _, n := range names {
		want[n] = true
	}
	out := make([]contract.Descriptor, 0, len(names))
	for _, c := range a.Capabilities {
		if want[c.Descriptor.Name] {
			out = append(out, c.Descriptor)
		}
	}
	return out
}

// tableCount 表数＝各 Descriptor.Owns 的并集大小（同一张表重复声明只算一次）。
func tableCount(descs []contract.Descriptor) int {
	seen := map[string]bool{}
	for _, d := range descs {
		for _, t := range d.Owns {
			seen[t] = true
		}
	}
	return len(seen)
}

// migrationCount 迁移数＝各 Descriptor.Migrations 里 migrations/mysql 下的 .sql 文件数。
// postgres 与 mysql 迁移同名同数（`check-capabilities` 同一口径），数一遍不重复计数。
func migrationCount(descs []contract.Descriptor) (int, error) {
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

// routeCount 把模块注册到裸 *gin.Engine 后统计 r.Routes()。运行时组合根对声明
// MountProtected 的能力用 router.Group("", protected...) 挂载，空 relativePath 的 Group
// 不改变任何路径（只追加 handler），因此裸引擎上的路由集合与实际启动逐条一致 —— 这里量
// 的是「注册了哪些路由」，不涉及监听、中间件或鉴权语义。
func routeCount(modules []contract.Module) int {
	r := gin.New()
	for _, m := range modules {
		m.RegisterHTTP(r)
	}
	return len(r.Routes())
}

// buildSize 构建该形态入口并返回产物字节数。
func buildSize(root, name string) (int64, error) {
	dir, err := os.MkdirTemp("", "jimu-compose-report-")
	if err != nil {
		return 0, err
	}
	defer func() { _ = os.RemoveAll(dir) }()

	out := filepath.Join(dir, "jimu-"+name)
	cmd := exec.CommandContext(context.Background(), "go", "build", "-o", out, "./profiles/"+name)
	cmd.Dir = root
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return 0, fmt.Errorf("go build ./profiles/%s: %w: %s", name, err, strings.TrimSpace(stderr.String()))
	}
	info, err := os.Stat(out)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// closureSize 统计该形态 import 闭包中本模块（jimu/...）非 _test.go 的 .go 文件数与行数。
// go/packages 在只请求名称/文件/import 图时等价于 go list -deps，不做类型检查。
func closureSize(root, name string) (files, lines int, err error) {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedImports | packages.NeedDeps,
		Dir:  root,
	}
	pkgs, err := packages.Load(cfg, "./profiles/"+name)
	if err != nil {
		return 0, 0, err
	}
	if len(pkgs) == 0 {
		return 0, 0, fmt.Errorf("no packages matched ./profiles/%s", name)
	}

	var loadErrs []string
	seenFile := map[string]bool{}
	packages.Visit(pkgs, func(p *packages.Package) bool {
		for _, e := range p.Errors {
			loadErrs = append(loadErrs, e.Error())
		}
		if !strings.HasPrefix(p.PkgPath, modulePath+"/") {
			return true
		}
		for _, f := range p.GoFiles {
			if strings.HasSuffix(f, "_test.go") || seenFile[f] {
				continue
			}
			seenFile[f] = true
			n, err := countLines(f)
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
		return 0, 0, fmt.Errorf("load package graph: %s", strings.Join(loadErrs, "; "))
	}
	return files, lines, nil
}

// countLines 计文件行数：换行符个数，末行无换行时补 1。
func countLines(path string) (int, error) {
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

// renderReport 输出 markdown 报告。输出必须只由实测值决定（无时间戳、无绝对路径），
// 这样 make compose-report 每次生成同一份文件，diff 只在形态真的变化时出现。
func renderReport(ms []Metrics, deps int) string {
	base := ms[0] // full 是基准列
	var b strings.Builder

	b.WriteString("# 形态编译面报告\n\n")
	b.WriteString("> 由 `make compose-report`（`tools/composereport`）生成，**请勿手工编辑**：改动形态组成后\n")
	b.WriteString("> 重跑该命令并提交本文件。设计依据见[能力可插拔设计](../design/2026-09-18-capability-plugins-design.md) §6.3 / §11。\n\n")

	b.WriteString("## 指标口径\n\n")
	b.WriteString("| 指标 | 口径 |\n|---|---|\n")
	b.WriteString("| 二进制 | `go build -o <tmp> ./profiles/<name>` 的产物大小 |\n")
	b.WriteString("| 路由数 | 形态解析集在裸 `gin.Engine` 上 `RegisterHTTP` 后的 `r.Routes()` 条数（不启动监听） |\n")
	b.WriteString("| 迁移数 | 各 `Descriptor.Migrations` 中 `migrations/mysql/*.sql` 的文件数（postgres 同名同数） |\n")
	b.WriteString("| 表数 | 各 `Descriptor.Owns` 的并集大小 |\n")
	b.WriteString("| 本仓 Go 文件 / 代码行 | `golang.org/x/tools/go/packages` 载入 `./profiles/<name>` 的 import 闭包，只统计本模块（`jimu/...`）的非 `_test.go` 文件 |\n")
	b.WriteString("| go.mod 直接依赖 | `go list -m -f '{{if not .Indirect}}{{.Path}}{{end}}' all` 的非空行数（不含主模块 `jimu` 自身） |\n\n")
	b.WriteString("「本仓闭包」严格大于「形态组成」：`user`/`auth` 直接 import 了 `outbox`/`queue`/`notification`/`ws` 的\n")
	b.WriteString("具体类型（`*outbox.Outbox`、`notification.Message`、`outbox.Event`），编译期会链上这些能力包，\n")
	b.WriteString("但装配期一个都不构造（详见 README「形态（profile）」的编译期脚注）。\n\n")

	b.WriteString("## 编译面\n\n")
	b.WriteString("| 形态 | 二进制 (MB) | 相对 full | 路由数 | 迁移数 | 表数 | 本仓 Go 文件 | 本仓代码行 |\n")
	b.WriteString("|---|---:|---:|---:|---:|---:|---:|---:|\n")
	for _, m := range ms {
		fmt.Fprintf(&b, "| `%s` | %s | %s | %d | %d | %d | %d | %d |\n",
			m.Profile, mb(m.BinaryBytes), percent(m.BinaryBytes, base.BinaryBytes),
			m.Routes, m.Migrations, m.Tables, m.Files, m.Lines)
	}
	b.WriteString("\n")

	b.WriteString("## 验收断言\n\n")
	b.WriteString("`go test ./tools/composereport/` 的 `TestMinimalCompiledSurfaceIsMateriallySmaller` 按下面的**实测关系**断言\n")
	b.WriteString("（不写死数字，内核膨胀或能力增减只会让真实的裁剪失效暴露出来）：\n\n")
	if base.Profile == "full" {
		if min, ok := findByProfile(ms, "minimal"); ok {
			fmt.Fprintf(&b, "- 二进制：`minimal` 是 `full` 的 %s（要求 ≤ 85%%）\n",
				percent(min.BinaryBytes, base.BinaryBytes))
			fmt.Fprintf(&b, "- 路由数：`minimal` %d < `full` %d\n", min.Routes, base.Routes)
			fmt.Fprintf(&b, "- 表数：`minimal` %d < `full` %d\n", min.Tables, base.Tables)
			fmt.Fprintf(&b, "- 本仓 Go 文件：`minimal` %d < `full` %d\n", min.Files, base.Files)
			fmt.Fprintf(&b, "- 本仓代码行：`minimal` %d < `full` %d\n", min.Lines, base.Lines)
		}
	}
	b.WriteString("\n")

	b.WriteString("## 层②边界：go.mod 直接依赖\n\n")
	fmt.Fprintf(&b, "五个形态的 go.mod 直接依赖数**逐形态完全相同**（各 %d 个）。这不是漏测：Go 的依赖裁剪\n", deps)
	b.WriteString("作用于整个 module —— `go.mod`/`go.sum` 描述 module 而非包，profile 入口包只改变**编进二进制的\n")
	b.WriteString("包与符号集合**，不改变 module 依赖图。真正让 `go.mod` 变小的是层①（`jimu new` 生成专属 module\n")
	b.WriteString("并 `go mod tidy`），不是层②。设计原文见 §11「层②的边界」。\n\n")

	b.WriteString("## 形态组成（解析集）\n\n")
	b.WriteString("| 形态 | 装配的能力（按装配顺序） |\n|---|---|\n")
	for _, m := range ms {
		fmt.Fprintf(&b, "| `%s` | %s |\n", m.Profile, strings.Join(m.Capabilities, " "))
	}
	return b.String()
}

func findByProfile(ms []Metrics, name string) (Metrics, bool) {
	for _, m := range ms {
		if m.Profile == name {
			return m, true
		}
	}
	return Metrics{}, false
}

// mb 以 MB（10^6 字节）呈现二进制大小，保留一位小数。
func mb(n int64) string {
	return fmt.Sprintf("%.1f", float64(n)/1e6)
}

// percent 返回 part 占 base 的百分比，保留一位小数。
func percent(part, base int64) string {
	if base == 0 {
		return "—"
	}
	return fmt.Sprintf("%.1f%%", 100*float64(part)/float64(base))
}

// directDeps 返回 go.mod 的直接依赖数（`go list -m` 输出里排除主模块自身）。
func directDeps(root string) (int, error) {
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
