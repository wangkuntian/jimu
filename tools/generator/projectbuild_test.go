package generator

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 本文件收敛「生成项目构建/类型检查」测试的两件事（Fix round 4）：
//
//  1. GOCACHE 一律用**随测试自动删除**的专用目录。共享的 $TMPDIR 构建缓存曾在 25 次并行链接下
//     涨到 25G 并把磁盘写满（no space left on device），还写坏了默认缓存；这里给唯一入口
//     newTestGoCache 并在 TestMain 做收尾自检。
//  2. 「全树不得残留框架 module path」的精确判据：protobuf 的 raw descriptor 与 gRPC Metadata
//     里的 proto 文件名是**必须保留**的（逐字节/与编译进二进制的描述符一致），见
//     allowedProtoLiteral。

// newProjectForTest 是本仓测试调用 NewProject 的唯一入口：一律跳过 ⑨ 自检（NoSelfCheck）。
//
// 动机与上一条约束同源：NewProject 的自检会走**环境里的** GOCACHE，而本包的构建类测试一律用
// 专用 GOCACHE 自行 build/vet（否则 25 次链接会把共享缓存写爆）。真实的自检路径（默认跑
// go build + checkcapabilities）由 `jimu new` 的端到端验收与 `make check-templates` 覆盖，
// 不在这里重复。
func newProjectForTest(t *testing.T, opts NewOptions) (*Result, error) {
	t.Helper()
	opts.NoSelfCheck = true
	return NewProject(opts)
}

// heavyMatrixEnv 门控「真实生成 + 构建/测试」的重型矩阵：未设置时默认路径只跑轻量的生成/渲染/文件断言，
// CI 的 `Scaffold Matrix` job 设 JIMU_HEAVY_MATRIX=1 跑满（见 requireHeavyMatrix）。
const heavyMatrixEnv = "JIMU_HEAVY_MATRIX"

// scaffoldShardEnv 只对两条选区网生效；本地未设置时仍执行全部选区。
const scaffoldShardEnv = "JIMU_SCAFFOLD_SHARD"

// testGoCacheEnv 把测试用的 GOCACHE 指到一个**可跨运行复用**的目录（见 newTestGoCache）；
// CI 的 Scaffold Matrix job 靠它让冷缓存只在首次付出代价 —— 前提是生成项目的构建都带 -trimpath
// （见 trimpathGoflags）：否则路径相关条目会无界增长，缓存既不收敛也换不来时间。
const testGoCacheEnv = "JIMU_TEST_GOCACHE"

// heavyBuildConcurrency 是重型矩阵里「同时跑几个真实生成项目的 go build/vet/test/run」的上限：
// 取值 = CI runner 的 vCPU 数（ubuntu-latest 标准 runner 为 4 核）。
//
// 定死在 2 会把 4 核用掉一半：历史全量选区构建网的子用例耗时之和约 1088s，`sem=2` 下墙钟约 560s，
// 而这段墙钟占了整个 Scaffold Matrix job（861s）的 65%。并发链接/编译对 CPU、内存与磁盘压力都大
// （历史事故：共享构建缓存涨到 25G 把磁盘写满，故构建测试一律用 newTestGoCache 的专用缓存），
// 所以这个数是**显式的上限**而不是无限并发；如果重型 job 出现内存/磁盘压力，先降它。
const heavyBuildConcurrency = 4

// scaffoldShardIndices 按选区在原始清单中的位置分片，确保每个选区恰好由一个 shard 执行。
func scaffoldShardIndices(spec string, count int) ([]int, error) {
	shard, total := 1, 1
	if spec != "" {
		parts := strings.Split(spec, "/")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid scaffold shard %q: expected N/TOTAL", spec)
		}
		var err error
		shard, err = strconv.Atoi(parts[0])
		if err != nil {
			return nil, fmt.Errorf("invalid scaffold shard %q: %w", spec, err)
		}
		total, err = strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("invalid scaffold shard %q: %w", spec, err)
		}
	}
	if count < 1 || shard < 1 || total < 1 || shard > total || total > count {
		return nil, fmt.Errorf("invalid scaffold shard %q for %d cases", spec, count)
	}
	indices := make([]int, 0, (count+total-1)/total)
	for i := 0; i < count; i++ {
		if i%total == shard-1 {
			indices = append(indices, i)
		}
	}
	return indices, nil
}

// requireHeavyMatrix 是本包重型用例的**唯一**门控入口：`-short` 或未设 `JIMU_HEAVY_MATRIX=1` 都跳过
// （只认字面量 `1`；`0`/`false`/空一律算关闭）。判据是「这条用例会把生成项目整棵树真的 shell out
// 到 go build/vet/test/run」——纯生成/渲染/文件断言等便宜用例不门控。
func requireHeavyMatrix(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("重型脚手架矩阵在 -short 下跳过")
	}
	if os.Getenv(heavyMatrixEnv) != "1" {
		t.Skip("真实生成 + 构建/测试矩阵需要 JIMU_HEAVY_MATRIX=1（CI 的 Scaffold Matrix job）")
	}
}

// newTestGoCache 返回本次构建测试使用的 GOCACHE：
//
//	JIMU_TEST_GOCACHE 非空 → 原样用它（**不删除**）：CI 的 Scaffold Matrix job 把它指向 actions/cache
//	                         的目录，跨运行复用构建产物；
//	否则                  → <t.TempDir()>/gocache（随测试删除），避免往共享缓存堆链接产物
//	                         （曾涨到 25G 写满磁盘并写坏默认缓存）。
func newTestGoCache(t *testing.T) string {
	t.Helper()
	cache := filepath.Join(t.TempDir(), "gocache")
	if dir := os.Getenv(testGoCacheEnv); dir != "" {
		cache = dir
	}
	require.NoError(t, os.MkdirAll(cache, 0o755))
	return cache
}

// assertProjectBuilds / assertProjectVets 在生成项目里跑 go build / go vet，失败即带输出报错。
func assertProjectBuilds(t *testing.T, dir, cache string) {
	runGoInProject(t, dir, cache, "build", "./...")
}
func assertProjectVets(t *testing.T, dir, cache string) {
	runGoInProject(t, dir, cache, "vet", "./...")
}

// trimpathGoflags 返回带 -trimpath 的 GOFLAGS（在用户既有值之后追加，已有则原样返回）。
//
// 生成的临时项目每次落在不同路径；不加 -trimpath 时绝对路径进入 build action，同一份内容在不同
// 路径下命中不了缓存 —— JIMU_TEST_GOCACHE 复用失效，且缓存体积随每次重跑无界增长（实测：全矩阵
// 无 -trimpath 冷 452s/5.5G、同缓存重跑仍 +300MB；加 -trimpath 后 400s→335s、2378MB→2391MB 收敛）。
func trimpathGoflags() string {
	cur := os.Getenv("GOFLAGS")
	switch {
	case cur == "":
		return "-trimpath"
	case strings.Contains(cur, "-trimpath"):
		return cur
	default:
		return cur + " -trimpath"
	}
}

// runGoInProject 在生成项目里执行一个 go 子命令（GOWORK=off + 指定 GOCACHE + -trimpath），失败即报错。
func runGoInProject(t *testing.T, dir, cache string, args ...string) {
	t.Helper()
	output, err := runGoInProjectOutput(t, dir, cache, args...)
	require.NoError(t, err, "生成项目 go %s 失败:\n%s", strings.Join(args, " "), output)
}

// runGoInProjectOutput 同上，但把输出与错误一并返回（供「允许登记失败」的测试自行判定）。
//
// env 用**最小显式 env**（serverEnvBaseForTest），绝不 `append(os.Environ(), …)`：生成项目里含有本仓
// 复制过去的集成测试，它们按「库可达就真跑、否则跳过」（testutil.SkipUnlessMysql）决定行为。若继承外界
// 的 DB_*，这些用例会从「跳过」变成「真跑」，而重型矩阵会**并行**生成十几个项目、各自对同一个
// jimu_test 库跑 goose 迁移 → 随机撞 `Error 1060 Duplicate column name 'totp_secret'`。
// CI 的 tag 发布 job（release.yml 的 Quality Gate）正好注入了指向真实 MariaDB 的 DB_*，于是整片
// TestGeneratedProjectTestTreeIsGreen 全红；ci-scaffold.yml 不注入 DB_*，所以 PR 路径一直绿。
// 见 TestServerEnvForTestCarriesNoAppConfigKeys。
func runGoInProjectOutput(t *testing.T, dir, cache string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	cmd.Env = serverEnvBaseForTest(cache)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// staleModulePathConst 是「裸 jimu 作为 module path」的唯一形态：tools/{composereport,
// checkcapabilities} 的 `const modulePath = "jimu"`。其它位置的裸 `"jimu"` 是**框架运行期名字**
// （观测 namespace/service name、JWT issuer、DB user/database 默认值、缓存 key 前缀、Kafka topic、
// CLI `Use: "jimu"`、TOTP issuer…），第 1 节裁定 3 明确「不改框架自己的名字」，故不在判据内。
var staleModulePathConst = regexp.MustCompile(`modulePath\s*=\s*"jimu"`)

// assertNoStaleModulePath 精确判据（Fix round 4，替代只查 `"jimu/` 的宽松版本）：
// 生成树里不得残留框架 module path 的**两种**形态 —— `"jimu/…`（import 前缀）与
// `modulePath = "jimu"`（度量工具里的 module path 常量）；**显式放行** protobuf rawDesc 与
// gRPC Metadata 里必须保留的 proto 文件名。这样既能在含 grpc 的选择上跑（machine），也不会因为
// `Metadata: "jimu/v1/ping.proto"` 这类**应当保留**的字面量而误报。
func assertNoStaleModulePath(t *testing.T, root string) {
	t.Helper()
	var stale []string
	require.NoError(t, filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".git") {
				return fs.SkipDir
			}
			return nil
		}
		if !isRewriteTarget(d.Name()) {
			return nil
		}
		content, rerr := os.ReadFile(p)
		if rerr != nil {
			return rerr
		}
		if !utf8.Valid(content) {
			return nil
		}
		rel := relPath(root, p)
		if rel == markerFile {
			return nil
		}
		inRawDesc := false
		for i, line := range strings.Split(string(content), "\n") {
			wasRawDesc := inRawDesc
			if strings.Contains(line, "_rawDesc = ") {
				inRawDesc, wasRawDesc = true, true
			}
			staleHere := strings.Contains(line, `"jimu/`) || staleModulePathConst.MatchString(line)
			if staleHere && !allowedProtoLiteral(line, wasRawDesc) {
				stale = append(stale, fmt.Sprintf("%s:%d: %s", rel, i+1, strings.TrimSpace(line)))
			}
			// rawDesc 是一个多行字符串拼接：最后一行不以 "+" 结尾。
			if inRawDesc && !strings.HasSuffix(strings.TrimSpace(line), "+") {
				inRawDesc = false
			}
		}
		return nil
	}))
	assert.Empty(t, stale, "生成树里残留了未重写的框架 module path（仅 protobuf rawDesc/Metadata 允许）：%v", stale)
}

// allowedProtoLiteral 判定一行里出现的 `"jimu/` / `"jimu"` 是否属于必须保留的 protobuf 字面量：
//
//	protobuf raw descriptor 常量（长度前缀，逐字节不可改）；
//	注释（如 `// source: proto/jimu/v1/userinfo.proto`）；
//	服务元数据里的 proto 文件名（`Metadata: "jimu/v1/ping.proto"`、
//	`Metadata: "proto/jimu/v1/userinfo.proto"`）—— 改了反而与编译进二进制的描述符不一致。
func allowedProtoLiteral(line string, inRawDesc bool) bool {
	if inRawDesc {
		return true
	}
	if strings.HasPrefix(strings.TrimSpace(line), "//") {
		return true
	}
	return strings.Contains(line, "Metadata:")
}

// TestGoBuildCacheIsTestScoped 收尾自检：构建测试必须用 newTestGoCache（默认专用、随测试删除；
// 设了 JIMU_TEST_GOCACHE 则原样复用该目录），不得再出现共享的 $TMPDIR/jimu-go-build-cache。
func TestGoBuildCacheIsTestScoped(t *testing.T) {
	cache := newTestGoCache(t)
	require.DirExists(t, cache)
	if dir := os.Getenv(testGoCacheEnv); dir != "" {
		assert.Equal(t, dir, cache, "JIMU_TEST_GOCACHE 必须原样生效（CI 靠它复用构建缓存）")
		return
	}
	assert.Equal(t, "gocache", filepath.Base(cache), "专用 GOCACHE 必须是 <t.TempDir()>/gocache")
	t.Logf("专用 GOCACHE 根目录 %s 将由 testing 在本用例结束时删除", filepath.Dir(cache))
}

// TestNoTestUsesSharedGoBuildCache 静态收尾自检：除本文件外，任何测试都不得引用共享构建缓存
// 路径（它曾涨到 25G 写满磁盘）。这是对「以后有人又写回共享缓存」的硬约束。
func TestNoTestUsesSharedGoBuildCache(t *testing.T) {
	files, err := filepath.Glob("*_test.go")
	require.NoError(t, err)
	require.NotEmpty(t, files)
	for _, name := range files {
		if name == "projectbuild_test.go" {
			continue // 本文件是「禁止引用」规则的载体，说明文字里会出现该名字
		}
		content, rerr := os.ReadFile(name)
		require.NoError(t, rerr)
		assert.NotContains(t, string(content), `"`+"jimu-go-build-cache"+`"`,
			"%s 仍在引用共享构建缓存，请改用 newTestGoCache", name)
	}
}

// TestMain 收尾自检（Fix round 4）：跑完后不得留下共享构建缓存；若发现（历史遗留）就打印并清除，
// 避免下一次再被写满磁盘。
func TestMain(m *testing.M) {
	legacy := filepath.Join(os.TempDir(), "jimu-go-build-cache")
	if _, err := os.Stat(legacy); err == nil {
		fmt.Fprintf(os.Stderr, "[收尾自检] 发现历史遗留的共享构建缓存 %s（%.1f MB），正在清除；构建测试请用 newTestGoCache\n",
			legacy, float64(dirSize(legacy))/(1024*1024))
		_ = os.RemoveAll(legacy)
	}
	code := m.Run()
	if _, err := os.Stat(legacy); err == nil {
		fmt.Fprintf(os.Stderr, "[收尾自检] 失败：共享构建缓存 %s 仍然存在\n", legacy)
		code = 1
	}
	os.Exit(code)
}

// dirSize 统计目录占用（收尾自检的提示信息用）。
func dirSize(root string) int64 {
	var total int64
	_ = filepath.WalkDir(root, func(_ string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if info, ierr := d.Info(); ierr == nil {
			total += info.Size()
		}
		return nil
	})
	return total
}

// TestStaleModulePathPredicateUnits 反向自检：判据必须能抓到未重写的 import 与
// `modulePath = "jimu"`，且不误报 protobuf 字面量与框架运行期名字。
func TestStaleModulePathPredicateUnits(t *testing.T) {
	assert.True(t, staleModulePathConst.MatchString(`const modulePath = "jimu"`), "未重写的 modulePath 常量必须被抓")
	assert.False(t, staleModulePathConst.MatchString(`const modulePath = "example.com/proj"`))
	assert.False(t, staleModulePathConst.MatchString(`const otherConst = "jimu"`), "只认 modulePath 这个声明名")

	// 必须抓：import 形态。
	assert.False(t, allowedProtoLiteral("\t\"jimu/internal/b\"", false))
	assert.False(t, allowedProtoLiteral(`const modulePath = "jimu"`, false))
	// 必须放行：protobuf rawDesc / 注释 / gRPC Metadata。
	assert.True(t, allowedProtoLiteral(`	"\x1cproto/jimu/v1/userinfo.proto\x12\ajimu.v1\")\n" +`, true))
	assert.True(t, allowedProtoLiteral(`// source: proto/jimu/v1/userinfo.proto`, false))
	assert.True(t, allowedProtoLiteral(`	Metadata: "jimu/v1/ping.proto",`, false))
	// 必须放行：框架运行期名字（裁定 3）——它们既不是 import 也不是 modulePath 常量。
	assert.False(t, staleModulePathConst.MatchString(`	Namespace: "jimu",`))
	assert.False(t, staleModulePathConst.MatchString(`	Use:   "jimu",`))
	assert.False(t, staleModulePathConst.MatchString(`	Issuer: "jimu",`))
	assert.False(t, strings.Contains(`	user: "jimu"`, `"jimu/`))
}
