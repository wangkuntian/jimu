// Command composereport 生成各形态（profile）的「编译面」报告。
//
// 报告回答「层②（形态选点）究竟改变了什么」：二进制大小、路由数、迁移数与表数逐
// 形态实测（二进制与闭包口径 = `./cmd/server` + 该形态 overlay，即出货二进制），本仓 import
// 闭包的代码行数/文件数按模块内包统计；同时写明 go.mod 直接依赖
// 数在各形态间**完全相同**（Go 的依赖裁剪作用于整个 module，设计 §6.3/§11 的层②边界）。
// 生成物 docs/profiles/compose-report.md 入库，供 CI 归档对比。
//
// 度量原语（路由数 / 迁移数 / 表数 / 闭包文件数与代码行 / 行数 / 直接依赖数）由
// tools/internal/projectmetrics 提供 —— 它与 `jimu new --report` 共用同一份口径，两处不会漂移。
//
// 全部指标不连库、不连 Redis、不启动监听：路由数在裸 *gin.Engine 上注册后统计。
package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"

	"jimu/internal/profiles/registry"
	"jimu/tools/internal/profileoverlay"
	"jimu/tools/internal/projectmetrics"
)

// outputPath 报告生成位置（相对仓库根）。
const outputPath = "docs/profiles/compose-report.md"

// modulePath 本模块的 import 前缀：代码量只统计本模块的包（形态差异全部来自本仓代码，
// 第三方依赖的代码量会把信号淹没，那部分由二进制大小衡量）。
const modulePath = "jimu"

// Metrics 是 projectmetrics.Metrics 的别名：库与报告的渲染/测试共用同一类型。
type Metrics = projectmetrics.Metrics

func main() {
	check := flag.Bool("check", false, "只校验入库报告与本次实测一致（平台无关部分），不写文件")
	flag.Parse()

	root, err := os.Getwd()
	if err != nil {
		fail(err)
	}
	ms, err := measureAll(root, nil)
	if err != nil {
		fail(err)
	}
	deps, err := projectmetrics.DirectDeps(root, modulePath)
	if err != nil {
		fail(err)
	}
	rendered := renderReport(ms, deps)
	out := filepath.Join(root, outputPath)

	// 二进制大小是**平台相关**的（同一份代码在 darwin/arm64 与 linux/amd64 上不同），所以入库报告
	// 里那一列无法逐字节门禁 —— 这里把它当**归档数据**打到日志，方便发布/排查时对比。
	printBinarySizes(ms)

	if *check {
		if err := checkCommitted(out, rendered); err != nil {
			fail(err)
		}
		fmt.Printf("✅ compose-report-check: %s 与实测一致（二进制列为平台相关，已掩码后比对）\n", outputPath)
		return
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		fail(err)
	}
	if err := os.WriteFile(out, []byte(rendered), 0o644); err != nil {
		fail(err)
	}
	fmt.Printf("✅ compose-report: %s\n", outputPath)
}

// printBinarySizes 把各形态的二进制大小与相对 full 的比例打到 stdout：入库报告不门禁这一列，
// 但 CI 日志要留档（§9 的「报告进 CI 归档对比」）。
func printBinarySizes(ms []Metrics) {
	var base int64
	for _, m := range ms {
		if m.Profile == "full" {
			base = m.BinaryBytes
		}
	}
	for _, m := range ms {
		if base > 0 && m.Profile != "full" {
			fmt.Printf("   binary %-11s %s MB (%s of full)\n", m.Profile, mb(m.BinaryBytes), percent(m.BinaryBytes, base))
			continue
		}
		fmt.Printf("   binary %-11s %s MB\n", m.Profile, mb(m.BinaryBytes))
	}
}

// volatileCellsRe / volatileSizeRe / volatileRatioRe 命中报告里**平台相关**的数字：多形态表的
// 「二进制 (MB) | 相对 full」两列、单形态表的「二进制 (MB)」列，以及验收段里 `minimal` 相对 `full`
// 的比例。
var (
	volatileCellsRe = regexp.MustCompile("(\\| `[^`]+` \\| )[0-9.]+ \\| [0-9.]+% ")
	volatileSizeRe  = regexp.MustCompile("(\\| `[^`]+` \\| )[0-9.]+ ")
	volatileRatioRe = regexp.MustCompile("是 `full` 的 [0-9.]+%")
)

// maskVolatileCells 把两处平台相关的数字换成占位符。**必须对「入库报告」与「本次渲染」同时施加**：
// 掩码规则本身不精确也只影响两边同等位置，比对仍然成立；而路由/迁移/表/文件数/代码行/重型依赖
// 这些平台无关的列一个都不会被掩掉。
func maskVolatileCells(text string) string {
	text = volatileCellsRe.ReplaceAllString(text, "${1}— | — ")
	text = volatileSizeRe.ReplaceAllString(text, "${1}— ")
	return volatileRatioRe.ReplaceAllString(text, "是 `full` 的 —%")
}

// checkCommitted 比对入库报告与本次渲染的**平台无关部分**；不一致时报出首个差异行。
func checkCommitted(committedPath, rendered string) error {
	committed, err := os.ReadFile(committedPath)
	if err != nil {
		return fmt.Errorf("读取入库报告 %s: %w", filepath.ToSlash(committedPath), err)
	}
	want := strings.Split(maskVolatileCells(rendered), "\n")
	got := strings.Split(maskVolatileCells(string(committed)), "\n")
	for i := 0; i < len(want) || i < len(got); i++ {
		w, g := "", ""
		if i < len(want) {
			w = want[i]
		}
		if i < len(got) {
			g = got[i]
		}
		if w != g {
			return fmt.Errorf("%s 与实测不一致（形态编译面发生漂移），第 %d 行：\n  入库: %s\n  实测: %s\n👉 若漂移是有意的，重新生成并提交：make compose-report", outputPath, i+1, g, w)
		}
	}
	return nil
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "❌ compose-report:", err)
	os.Exit(1)
}

// measureAll 按 registry.Names() 顺序实测形态；profiles 非空时只实测这些形态（保持 registry 顺序）。
//
// 形态子集是给 CI 的 race 分片用的：这条用例在 `-race` 下的进程内度量（每个形态一次
// `packages.Load` 全依赖图 + 全闭包行数统计）在 4 vCPU runner 上要几分钟，而它只是 tools 分片里的
// 一个包 —— 按形态切成多片后每片只度量自己那部分（见 scripts/race_shards.sh；跨形态关系仍由
// 未设子集时的完整度量断言）。
//
// 二进制并行构建（互不共享状态），随后逐形态顺序探测：ProbeAssembly 会临时切换
// 工作目录以吸收构造期相对路径副作用，因此不能并发。
func measureAll(root string, profiles []string) ([]Metrics, error) {
	names := registry.Names()
	asms := registry.All()
	if len(profiles) > 0 {
		picked, err := pickProfiles(names, profiles)
		if err != nil {
			return nil, err
		}
		names = picked
	}

	sizes := make([]int64, len(names))
	errs := make([]error, len(names))
	var wg sync.WaitGroup
	for i, name := range names {
		wg.Add(1)
		go func(i int, name string) {
			defer wg.Done()
			sizes[i], errs[i] = buildSize(root, name)
		}(i, name)
	}
	wg.Wait()

	out := make([]Metrics, 0, len(names))
	for i, name := range names {
		if errs[i] != nil {
			return nil, errs[i]
		}
		// 闭包口径 = `./cmd/server` + 该形态的**构建期 overlay**（出货二进制），故这里取内存
		// overlay 交给共享原语；生成项目的报告是单形态提交态，传 nil。
		overlay, err := overlayForProfile(root, name)
		if err != nil {
			return nil, err
		}
		m, err := projectmetrics.Of(root, modulePath, asms[name], overlay)
		if err != nil {
			return nil, err
		}
		m.Profile = name
		m.BinaryBytes = sizes[i]
		out = append(out, m)
	}
	return out, nil
}

// pickProfiles 按 names 的顺序取出 wanted 里的形态。wanted 里出现未知形态即报错（fail-closed：
// 拼错形态名不该静默少测 —— CI 的分片正是用这个名字选形态）。
func pickProfiles(names, wanted []string) ([]string, error) {
	want := make(map[string]bool, len(wanted))
	for _, n := range wanted {
		want[n] = true
	}
	out := make([]string, 0, len(wanted))
	for _, n := range names {
		if want[n] {
			out = append(out, n)
			delete(want, n)
		}
	}
	if len(want) > 0 {
		return nil, fmt.Errorf("unknown profiles: %s", strings.Join(slices.Sorted(maps.Keys(want)), ", "))
	}
	return out, nil
}

// buildSize 构建唯一入口（该形态 overlay 下）并返回产物字节数。
//
// 两道约束要分清：overlay JSON 的**键**必须命中仓库里真实存在的被替换选点文件
// （`<root>/internal/profiles/active/assembly.go`）——键指向别处不会替换任何源文件，
// `go build` 会**静默按提交态（full）构建**（该失败被本文件的
// TestMinimalCompiledSurfaceIsMateriallySmaller 抓过一次）；而生成物 active.go 落到哪个目录
// 是自由的。共享包的 WriteFiles(root, profile) 把值固定在 `<root>/.overlay/<profile>/`，
// 因此这里只能传仓库根当 root —— 代价是 5 个形态的 overlay 常驻 gitignored 的 .overlay/。
func buildSize(root, name string) (int64, error) {
	dir, err := os.MkdirTemp("", "jimu-compose-report-")
	if err != nil {
		return 0, err
	}
	defer func() { _ = os.RemoveAll(dir) }()

	overlayPath, err := writeOverlay(root, name)
	if err != nil {
		return 0, err
	}
	out := filepath.Join(dir, "jimu-"+name)
	cmd := exec.CommandContext(context.Background(), "go", "build", "-overlay", overlayPath, "-o", out, "./cmd/server")
	cmd.Dir = root
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return 0, fmt.Errorf("go build -overlay < %s > ./cmd/server: %w: %s", name, err, strings.TrimSpace(stderr.String()))
	}
	info, err := os.Stat(out)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// writeOverlay 在仓库根下写该形态的构建期 overlay（`<root>/.overlay/<profile>/`）并返回
// JSON 路径供 `go build -overlay` 使用。模板、按形态隔离的目录与 JSON 形态都由共享包
// tools/internal/profileoverlay 提供，本报告不再持有副本。
func writeOverlay(root, name string) (string, error) {
	return profileoverlay.WriteFiles(root, name)
}

// overlayForProfile 返回把 active 选点替换为指定形态的内存 overlay（键值均为绝对路径）；
// 模板与路径口径都来自共享包 tools/internal/profileoverlay，本报告不再持有副本。
func overlayForProfile(root, profile string) (map[string][]byte, error) {
	return profileoverlay.ReplaceMap(root, profile)
}

// renderReport 输出 markdown 报告。输出必须只由实测值决定（无时间戳、无绝对路径），
// 这样 make compose-report 每次生成同一份文件，diff 只在形态真的变化时出现。
//
// 两个分支的区别是**被度量的对象**，不是风格：
//
//	>= 2 个形态（本仓：5 个）→ 横向对比报告（含「相对 full」归一化列、构建期选点叙述）；
//	恰好 1 个形态（`jimu new` 的生成项目）→ 单形态报告：没有可比对象、没有构建期叠加，
//	                                          主模块名就是该项目自己（modulePath 已被
//	                                          `jimu new` 的受控重写改成 --module 的值）。
//
// 本仓永远走多形态分支，故入库产物 docs/profiles/compose-report.md 逐字节不变（T8 裁定 15）。
func renderReport(ms []Metrics, deps int) string {
	if len(ms) == 1 {
		return renderSingleShapeReport(ms[0], deps)
	}
	base := ms[0] // full 是基准列
	var b strings.Builder

	b.WriteString("# 形态编译面报告\n\n")
	b.WriteString("> 由 `make compose-report`（`tools/composereport`）生成，**请勿手工编辑**：改动形态组成后\n")
	b.WriteString("> 重跑该命令并提交本文件。设计依据见[能力可插拔设计](../design/2026-09-18-capability-plugins-design.md) §6.3 / §11。\n\n")

	b.WriteString("## 指标口径\n\n")
	b.WriteString("| 指标 | 口径 |\n|---|---|\n")
	b.WriteString("| 二进制 | `go build -overlay=<该形态> -o <tmp> ./cmd/server` 的产物大小 |\n")
	b.WriteString("| 路由数 | 形态解析集在裸 `gin.Engine` 上 `RegisterHTTP` 后的 `r.Routes()` 条数（不启动监听） |\n")
	b.WriteString("| 迁移数 | **迁移集**（形态声明集 ∪ schema 依赖，与 `PROFILE=<name> jimu migrate` 同一口径）各 `Descriptor.Migrations` 中 `migrations/mysql/*.sql` 的文件数（postgres 同名同数） |\n")
	b.WriteString("| 表数 | **迁移集**各 `Descriptor.Owns` 的并集大小（口径同迁移数） |\n")
	b.WriteString("| 本仓 Go 文件 / 代码行 | `golang.org/x/tools/go/packages` 载入 `./cmd/server` 在该形态 overlay 下的 import 闭包，只统计本模块（`jimu/...`）的非 `_test.go` 文件 |\n")
	b.WriteString("| 重型依赖 | 同一闭包（含第三方包）命中 `tools/internal/heavydeps` 前缀表的展示名，`-` 表示零 |\n")
	b.WriteString("| go.mod 直接依赖 | `go list -m -f '{{if not .Indirect}}{{.Path}}{{end}}' all` 的非空行数（不含主模块 `jimu` 自身） |\n\n")
	b.WriteString("形态由 `internal/profiles/active` 的**构建期 overlay** 决定：`tools/profileoverlay` 把该选点文件\n")
	b.WriteString("换成「只选一个形态」的版本，`go build ./cmd/server` 因此只编进该形态的能力与驱动。\n\n")
	b.WriteString("「本仓闭包」严格大于「形态组成」：`user`/`auth` 直接 import 了 `outbox`/`queue`/`notification`/`ws` 的\n")
	b.WriteString("具体类型（`*outbox.Outbox`、`notification.Message`、`outbox.Event`），编译期会链上这些能力包，\n")
	b.WriteString("但装配期一个都不构造（详见 README「形态（profile）」的编译期脚注）。\n\n")

	b.WriteString("## 编译面\n\n")
	b.WriteString("| 形态 | 二进制 (MB) | 相对 full | 路由数 | 迁移数 | 表数 | 本仓 Go 文件 | 本仓代码行 | 重型依赖 |\n")
	b.WriteString("|---|---:|---:|---:|---:|---:|---:|---:|---|\n")
	for _, m := range ms {
		fmt.Fprintf(&b, "| `%s` | %s | %s | %d | %d | %d | %d | %d | %s |\n",
			m.Profile, mb(m.BinaryBytes), percent(m.BinaryBytes, base.BinaryBytes),
			m.Routes, m.Migrations, m.Tables, m.Files, m.Lines, heavyDepsCell(m.HeavyDeps))
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

// renderSingleShapeReport 是**单形态**项目（`jimu new` 的产物）的报告：只有一个形态，因此
// 不写「相对 full」归一化列、不提构建期叠加、不引用本仓 README/测试（生成项目里都不存在），
// 主模块名用 modulePath（生成时已被改写成 --module 的值）。
func renderSingleShapeReport(m Metrics, deps int) string {
	var b strings.Builder
	b.WriteString("# 项目编译面报告\n\n")
	b.WriteString("> 由 `make compose-report`（`tools/composereport`）生成，**请勿手工编辑**：改动能力集后\n")
	b.WriteString("> 重跑该命令并提交本文件。本项目只有一个形态、在生成期固定，因此报告不与任何东西比较。\n\n")

	b.WriteString("## 指标口径\n\n")
	b.WriteString("| 指标 | 口径 |\n|---|---|\n")
	b.WriteString("| 二进制 | `go build -o <tmp> ./cmd/server` 的产物大小 |\n")
	b.WriteString("| 路由数 | 本项目装配集在裸 `gin.Engine` 上 `RegisterHTTP` 后的 `r.Routes()` 条数（不启动监听） |\n")
	b.WriteString("| 迁移数 | **迁移集**（声明集 ∪ schema 依赖，与 `jimu migrate` 同一口径）各 `Descriptor.Migrations` 中 `migrations/mysql/*.sql` 的文件数（postgres 同名同数） |\n")
	b.WriteString("| 表数 | **迁移集**各 `Descriptor.Owns` 的并集大小（口径同迁移数） |\n")
	fmt.Fprintf(&b, "| 本模块 Go 文件 / 代码行 | `golang.org/x/tools/go/packages` 载入 `./cmd/server` 的 import 闭包，只统计本模块（`%s/...`）的非 `_test.go` 文件 |\n", modulePath)
	b.WriteString("| 重型依赖 | 同一闭包（含第三方包）命中 `tools/internal/heavydeps` 前缀表的展示名，`-` 表示零 |\n")
	fmt.Fprintf(&b, "| go.mod 直接依赖 | `go list -m -f '{{if not .Indirect}}{{.Path}}{{end}}' all` 的非空行数（不含主模块 `%s` 自身） |\n\n", modulePath)
	b.WriteString("形态在**生成期**固定：`internal/profiles/active/assembly.go` 直接指向本项目唯一形态，\n")
	b.WriteString("`go build ./cmd/server` 只编进本项目选定的能力与驱动（生成项目没有构建期叠加）。\n\n")
	b.WriteString("「本模块闭包」严格大于「装配集」：`user`/`auth` 直接 import 了 `outbox`/`queue`/`notification` 的\n")
	b.WriteString("具体类型（`*outbox.Outbox`、`notification.Message`、`outbox.Event`），编译期会链上这些能力包，\n")
	b.WriteString("但装配期一个都不构造。\n\n")

	b.WriteString("## 编译面\n\n")
	b.WriteString("| 形态 | 二进制 (MB) | 路由数 | 迁移数 | 表数 | 本模块 Go 文件 | 本模块代码行 | 重型依赖 |\n")
	b.WriteString("|---|---:|---:|---:|---:|---:|---:|---|\n")
	fmt.Fprintf(&b, "| `%s` | %s | %d | %d | %d | %d | %d | %s |\n",
		m.Profile, mb(m.BinaryBytes), m.Routes, m.Migrations, m.Tables, m.Files, m.Lines, heavyDepsCell(m.HeavyDeps))
	b.WriteString("\n")

	b.WriteString("## 装配集（解析集）\n\n")
	b.WriteString("| 装配的能力（按装配顺序） |\n|---|\n")
	fmt.Fprintf(&b, "| %s |\n", strings.Join(m.Capabilities, " "))
	b.WriteString("\n")

	b.WriteString("## 层②边界：go.mod 直接依赖\n\n")
	fmt.Fprintf(&b, "本项目的 go.mod 直接依赖数为 **%d** 个。这个数字与本项目装配了哪些能力无关：Go 的依赖裁剪\n", deps)
	b.WriteString("作用于整个 module —— `go.mod`/`go.sum` 描述 module 而非包，形态入口包只改变**编进二进制的\n")
	b.WriteString("包与符号集合**，不改变 module 依赖图。真正让 `go.mod` 变小的是生成期的能力选择与\n")
	b.WriteString("`jimu new` 的 `go mod tidy`。\n")
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

// heavyDepsCell 渲染重型依赖列：空集为 `-`，否则逗号分隔（HeavyDeps 已去重升序）。
func heavyDepsCell(deps []string) string {
	if len(deps) == 0 {
		return "-"
	}
	return strings.Join(deps, ", ")
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
