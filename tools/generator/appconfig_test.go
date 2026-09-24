package generator

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"jimu/internal/capabilities/catalog"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// frameworkRootForTest 返回框架仓根（从测试进程的 cwd 向上找 module jimu 的 go.mod）。
func frameworkRootForTest(t *testing.T) string {
	t.Helper()
	root, err := frameworkRoot()
	require.NoError(t, err)
	return root
}

// readRepoFile 读取框架仓内的文件（configs/app.yaml、deploy/helm/values.yaml 等蓝本）。
func readRepoFile(t *testing.T, rel string) []byte {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(frameworkRootForTest(t), filepath.FromSlash(rel)))
	require.NoError(t, err)
	return content
}

// scanLoadSectionKeys 扫描 dir（internal/capabilities）下各能力包的 config.go，
// 把 `config.LoadSection(dec, <段键>, …)` 的第二个实参（常量）解析成字符串，
// 返回 能力包名 → 段键（按源码顺序，去重）。解析不出来即测试失败（fail-closed）。
func scanLoadSectionKeys(t *testing.T, dir string) map[string][]string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	out := map[string][]string{}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		path := filepath.Join(dir, e.Name(), "config.go")
		if _, err := os.Stat(path); err != nil {
			continue
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, 0)
		require.NoError(t, err)
		consts := stringConsts(file)
		var keys []string
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "LoadSection" || len(call.Args) < 2 {
				return true
			}
			if pkg, ok := sel.X.(*ast.Ident); !ok || pkg.Name != "config" {
				return true
			}
			id, ok := call.Args[1].(*ast.Ident)
			require.Truef(t, ok, "%s: LoadSection 的段键必须是常量标识符，实际 %T", path, call.Args[1])
			value, ok := consts[id.Name]
			require.Truef(t, ok, "%s: 无法解析常量 %q（漂移护栏不会静默放过）", path, id.Name)
			if !slices.Contains(keys, value) {
				keys = append(keys, value)
			}
			return true
		})
		if len(keys) > 0 {
			out[e.Name()] = keys
		}
	}
	return out
}

// stringConsts 取一个 Go 文件里所有「字符串字面量常量」的名字 → 值。
func stringConsts(file *ast.File) map[string]string {
	out := map[string]string{}
	ast.Inspect(file, func(n ast.Node) bool {
		decl, ok := n.(*ast.GenDecl)
		if !ok || decl.Tok != token.CONST {
			return true
		}
		for _, spec := range decl.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for i, name := range vs.Names {
				if i >= len(vs.Values) {
					continue
				}
				lit, ok := vs.Values[i].(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					continue
				}
				value, err := strconv.Unquote(lit.Value)
				if err == nil {
					out[name.Name] = value
				}
			}
		}
		return true
	})
	return out
}

// kernelSectionsFromRepo 从 internal/config/config.go 的 `type Config struct` 字段
// mapstructure tag 派生内核段集合（与 KernelSections() 的运行时反射互证）。
func kernelSectionsFromRepo(t *testing.T) []string {
	t.Helper()
	path := filepath.Join(frameworkRootForTest(t), "internal", "config", "config.go")
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	require.NoError(t, err)
	var out []string
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.TYPE {
			continue
		}
		for _, spec := range gen.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok || ts.Name.Name != "Config" {
				continue
			}
			st, ok := ts.Type.(*ast.StructType)
			require.True(t, ok, "type Config 必须是 struct")
			for _, field := range st.Fields.List {
				if field.Tag == nil {
					continue
				}
				tag, err := strconv.Unquote(field.Tag.Value)
				require.NoError(t, err)
				name := reflect.StructTag(tag).Get("mapstructure")
				name, _, _ = strings.Cut(name, ",")
				if name == "" || name == "-" {
					continue
				}
				out = append(out, name)
			}
		}
	}
	require.NotEmpty(t, out, "未能从 config.go 解析出内核段")
	slices.Sort(out)
	return out
}

// TestSelfReadSectionsMatchTheFrameworkSources 是 selfReadSections（第三类段：能力自读但
// 未在 Descriptor.Configs 声明）的漂移护栏：唯一真源是各能力的 Load() 里的 LoadSection 实参。
// 扫描结果先**减去在 Descriptor.Configs 里已声明的段**：selfReadSections 只装「未声明」的段，
// 某个能力同时声明并自读同一段时应由第②类来源负责，不该在这里假红。
func TestSelfReadSectionsMatchTheFrameworkSources(t *testing.T) {
	root := frameworkRootForTest(t)
	scanned := scanLoadSectionKeys(t, filepath.Join(root, "internal", "capabilities"))
	got := undeclaredSections(scanned, declaredSections())
	assert.Equal(t, map[string][]string{
		"storage":      {"storage"},
		"retention":    {"retention"},
		"notification": {"email", "sms", "notification"},
	}, got)
	assert.Equal(t, got, selfReadSections, "selfReadSections 必须与能力实际 Load 的未声明段逐值一致")
}

// TestUndeclaredSectionsDropsConfigsDeclaredSegments 钉住上一条的语义：只按「同一能力自己声明的段」
// 做减法（别的能力声明同名段不影响本能力），被减掉的段整项消失、其余段顺序不变。
func TestUndeclaredSectionsDropsConfigsDeclaredSegments(t *testing.T) {
	scanned := map[string][]string{
		"captcha": {"captcha"},            // 声明 + 自读同一段 → 属于第②类
		"queue":   {"queue", "scheduler"}, // 声明 queue、自读 scheduler → 只剩 scheduler
		"storage": {"storage"},            // 自读且未声明 → 保留
	}
	declared := map[string][]string{"captcha": {"captcha"}, "queue": {"queue"}}
	assert.Equal(t, map[string][]string{
		"queue":   {"scheduler"},
		"storage": {"storage"},
	}, undeclaredSections(scanned, declared))
}

// declaredSections 返回 catalog 各能力在 Descriptor.Configs 里声明的段（能力名 → 段）。
func declaredSections() map[string][]string {
	out := map[string][]string{}
	for _, desc := range catalog.All() {
		if len(desc.Configs) == 0 {
			continue
		}
		for _, spec := range desc.Configs {
			out[desc.Name] = append(out[desc.Name], spec.Section)
		}
	}
	return out
}

// undeclaredSections 从扫描结果里减去 declared 中同一能力已声明的段（返回新 map，不改入参）。
func undeclaredSections(scanned, declared map[string][]string) map[string][]string {
	out := make(map[string][]string, len(scanned))
	for name, sections := range scanned {
		declaredHere := make(map[string]bool, len(declared[name]))
		for _, section := range declared[name] {
			declaredHere[section] = true
		}
		for _, section := range sections {
			if declaredHere[section] {
				continue
			}
			out[name] = append(out[name], section)
		}
		if len(out[name]) == 0 {
			delete(out, name)
		}
	}
	return out
}

// TestKernelSectionsMatchConfigStruct 是内核段的漂移护栏：KernelSections()（对
// internal/config.Config 的运行时反射）必须等于 config.go 里的 mapstructure tag 集合。
func TestKernelSectionsMatchConfigStruct(t *testing.T) {
	assert.ElementsMatch(t, kernelSectionsFromRepo(t), KernelSections())
	assert.Len(t, KernelSections(), 15)
}

// TestSectionsSourcesExplainEveryAppYAMLKey 钉住三类段的完整性：full 形态声明了全部 25 个
// 能力（catalog 18 ∪ Ungated 7），其段集合必须**恰好**等于蓝本 configs/app.yaml 的 28 个
// 顶层段 —— 任何一类来源漏掉，这里就少段（recon §5 的 5 段缺口）。
func TestSectionsSourcesExplainEveryAppYAMLKey(t *testing.T) {
	keys, _, err := SectionBlocks(readRepoFile(t, "configs/app.yaml"))
	require.NoError(t, err)
	set, err := ParseCapabilitySet("full", "", "")
	require.NoError(t, err)
	sections, err := SectionsFor(set)
	require.NoError(t, err)
	assert.Len(t, keys, 28)
	assert.ElementsMatch(t, keys, sections)
}

// TestSectionsForAddsSelfReadSectionsForUngatedCapabilities 覆盖 minimal：
// 声明段（auth）+ 自读段（email/sms/notification）+ 内核段，不含未选中能力的段。
func TestSectionsForAddsSelfReadSectionsForUngatedCapabilities(t *testing.T) {
	set, err := ParseCapabilitySet("minimal", "", "")
	require.NoError(t, err)
	got, err := SectionsFor(set)
	require.NoError(t, err)
	assert.Contains(t, got, "auth")
	assert.Contains(t, got, "email")
	assert.Contains(t, got, "sms")
	assert.Contains(t, got, "notification")
	assert.NotContains(t, got, "storage")
	assert.NotContains(t, got, "retention")
	assert.NotContains(t, got, "upload")
	assert.Contains(t, got, "http")
	assert.Len(t, got, 19)
	assert.True(t, slices.IsSorted(got), "段集合升序去重：%v", got)
}

// TestSectionsForWithQueueIncludesQueueAndScheduler：queue 的 Descriptor.Configs 声明两段。
func TestSectionsForWithQueueIncludesQueueAndScheduler(t *testing.T) {
	set, err := ParseCapabilitySet("", "user,access,queue", "app")
	require.NoError(t, err)
	got, err := SectionsFor(set)
	require.NoError(t, err)
	assert.Contains(t, got, "queue")
	assert.Contains(t, got, "scheduler")
	assert.NotContains(t, got, "auth")
	assert.Len(t, got, 17)
}

// TestAppConfigSectionBlocksRoundTripsRepoFiles：块切分必须无损 —— 拼接所有块逐字节等于源文件，
// 段序等于源文件顺序。这是「绝不静默丢内容」和「保留原文」的直接证据。
func TestAppConfigSectionBlocksRoundTripsRepoFiles(t *testing.T) {
	for _, tc := range []struct {
		rel  string
		keys int
	}{
		{"configs/app.yaml", 28},
		{"configs/app.prod.yaml", 27},
		{"deploy/helm/values.yaml", 24},
	} {
		src := readRepoFile(t, tc.rel)
		keys, blocks, err := SectionBlocks(src)
		require.NoError(t, err, tc.rel)
		assert.Len(t, keys, tc.keys, tc.rel)
		assert.Equal(t, string(src), strings.Join(blocks, ""), tc.rel)
		sorted := slices.Clone(keys)
		slices.Sort(sorted)
		assert.Len(t, slices.Compact(sorted), len(keys), "%s: 顶层键重复", tc.rel)
	}
}

// TestAppConfigSectionBlocksRejectsStrayTopLevelLine：无法识别的顶层行必须报错，不得静默丢内容。
func TestAppConfigSectionBlocksRejectsStrayTopLevelLine(t *testing.T) {
	_, _, err := SectionBlocks([]byte("http:\n  port: 8080\n---\n"))
	require.ErrorContains(t, err, "line 3")
	_, _, err = SectionBlocks([]byte("  http:\n    port: 8080\n"))
	require.ErrorContains(t, err, "line 1")
	// 顶层块序列项不是映射键：必须报错而不是被当成键名 "- name" 静默丢弃。
	_, _, err = SectionBlocks([]byte("- name: x\n"))
	require.ErrorContains(t, err, "line 1")
	_, _, err = SectionBlocks([]byte("http:\n  port: 8080\n- name: x\n"))
	require.ErrorContains(t, err, "line 3")
	_, _, err = SectionBlocks([]byte("-\n"))
	require.ErrorContains(t, err, "line 1")
}

// TestAppConfigRenderConfigsWriteFailureIsReported 覆盖 configs 渲染路径的故障注入：RenderConfigs 必须
// 走包内注入点 writeFile（module.go），写失败要作为错误返回而不是静默半成品。
func TestAppConfigRenderConfigsWriteFailureIsReported(t *testing.T) {
	originalWriteFile := writeFile
	t.Cleanup(func() { writeFile = originalWriteFile })
	sentinel := errors.New("disk full")
	writeFile = func(string, []byte, os.FileMode) error { return sentinel }

	set, err := ParseCapabilitySet("minimal", "", "")
	require.NoError(t, err)
	err = RenderConfigs(frameworkRootForTest(t), t.TempDir(), set)
	require.ErrorIs(t, err, sentinel)
	assert.Contains(t, err.Error(), "configs/app.yaml")
}

// TestRenderAppConfigPreservesSourceOrderAndIsIdempotent：keep 全集时逐字节等于源文件
// （顺序与注释都保持原文）；二次渲染幂等（capability add 的重渲染前提）。
func TestRenderAppConfigPreservesSourceOrderAndIsIdempotent(t *testing.T) {
	for _, rel := range []string{"configs/app.yaml", "configs/app.prod.yaml"} {
		src := readRepoFile(t, rel)
		keys, _, err := SectionBlocks(src)
		require.NoError(t, err)
		out, err := RenderAppConfig(src, keys)
		require.NoError(t, err)
		assert.Equal(t, string(src), string(out), rel)
		again, err := RenderAppConfig(out, keys)
		require.NoError(t, err)
		assert.Equal(t, string(src), string(again), rel)
	}
}

// TestAppConfigProdCoversEveryNonCapabilitySection：S6① 的资产不变量 —— app.prod.yaml 与
// app.yaml 段集合一致，只少 capabilities（prod 覆盖文件不再放大段集合）。顺序不必一致
// （两份蓝本的段序本就不同），段集合才是口径。
func TestAppConfigProdCoversEveryNonCapabilitySection(t *testing.T) {
	appKeys, _, err := SectionBlocks(readRepoFile(t, "configs/app.yaml"))
	require.NoError(t, err)
	prodKeys, _, err := SectionBlocks(readRepoFile(t, "configs/app.prod.yaml"))
	require.NoError(t, err)
	assert.ElementsMatch(t, slices.DeleteFunc(slices.Clone(appKeys), func(k string) bool { return k == "capabilities" }), prodKeys)
}

// TestRenderAppConfigKeepsOnlySelectedSections 用小型 fixture 覆盖段选择（含行内/独立注释、
// 引号键、段尾注释的归属）。
func TestRenderAppConfigKeepsOnlySelectedSections(t *testing.T) {
	src, err := os.ReadFile("testdata/appconfig/fixture.yaml")
	require.NoError(t, err)
	out, err := RenderAppConfig(src, []string{"http", "db", "capabilities", "queue", "scheduler"})
	require.NoError(t, err)
	assert.Contains(t, string(out), "\nqueue:\n")
	assert.Contains(t, string(out), "\nscheduler:\n")
	assert.Contains(t, string(out), "\"db\":\n")
	assert.Contains(t, string(out), "fixture 头部注释")
	assert.Contains(t, string(out), "# fixture 尾部注释")
	assert.Contains(t, string(out), "# 队列段前的独立注释组")
	assert.NotContains(t, string(out), "\nretention:\n")
	assert.NotContains(t, string(out), "\nstorage:\n")
	assert.NotContains(t, string(out), "\nnotification:\n")
	assert.NotContains(t, string(out), "# 保留段注释")
}

// TestAppConfigGoldenMinimal / …WithUserAccessQueue：黄金文件锁住「蓝本 → 段子集」的逐字节产物。
func TestAppConfigGoldenMinimal(t *testing.T) {
	set, err := ParseCapabilitySet("minimal", "", "")
	require.NoError(t, err)
	assertGoldenAppConfig(t, set, "testdata/appconfig/golden_minimal.yaml")
}

func TestAppConfigGoldenWithUserAccessQueue(t *testing.T) {
	set, err := ParseCapabilitySet("", "user,access,queue", "app")
	require.NoError(t, err)
	assertGoldenAppConfig(t, set, "testdata/appconfig/golden_with_user_access_queue.yaml")
}

func assertGoldenAppConfig(t *testing.T, set CapabilitySet, golden string) {
	t.Helper()
	sections, err := SectionsFor(set)
	require.NoError(t, err)
	out, err := RenderAppConfig(readRepoFile(t, "configs/app.yaml"), sections)
	require.NoError(t, err)
	want, err := os.ReadFile(golden)
	require.NoError(t, err)
	assert.Equal(t, string(want), string(out),
		"黄金文件 %s 与渲染结果不一致：蓝本 configs/app.yaml 或段选择规则变了，请核对后重生成", golden)
}

// TestValuesCapabilityKeysMatchRepo：values.yaml 的「能力同名键」映射表漂移护栏 ——
// 24 个顶层键里凡与某个能力同名者都必须在表里；表里也只能是这些键（S6②）。
func TestValuesCapabilityKeysMatchRepo(t *testing.T) {
	keys, _, err := SectionBlocks(readRepoFile(t, "deploy/helm/values.yaml"))
	require.NoError(t, err)
	assert.Len(t, keys, 24)
	set, err := ParseCapabilitySet("full", "", "")
	require.NoError(t, err)
	known := make(map[string]bool, len(set.Known))
	for _, name := range set.Known {
		known[name] = true
	}
	var repoCapabilityKeys []string
	for _, key := range keys {
		if known[key] {
			assert.Containsf(t, valuesCapabilityKeys, key, "values.yaml 的 %q 与能力同名，必须进 valuesCapabilityKeys", key)
			repoCapabilityKeys = append(repoCapabilityKeys, key)
		}
	}
	assert.ElementsMatch(t, []string{"audit", "storage", "auth"}, repoCapabilityKeys)
	assert.ElementsMatch(t, repoCapabilityKeys, slices.Collect(maps.Keys(valuesCapabilityKeys)))
}

// TestValuesSectionsKeepKernelKeysAndSelectedCapabilityKeys：内核键恒留（21 个）+ 选中能力同名键。
func TestValuesSectionsKeepKernelKeysAndSelectedCapabilityKeys(t *testing.T) {
	src := readRepoFile(t, "deploy/helm/values.yaml")
	keys, _, err := SectionBlocks(src)
	require.NoError(t, err)

	minimal, err := ParseCapabilitySet("minimal", "", "")
	require.NoError(t, err)
	keep, err := ValuesSections(src, minimal)
	require.NoError(t, err)
	assert.Len(t, keep, 22)
	assert.Contains(t, keep, "auth")
	assert.NotContains(t, keep, "audit")
	assert.NotContains(t, keep, "storage")
	assert.Equal(t, slices.DeleteFunc(slices.Clone(keys), func(k string) bool { return k == "audit" || k == "storage" }), keep)

	full, err := ParseCapabilitySet("full", "", "")
	require.NoError(t, err)
	keepAll, err := ValuesSections(src, full)
	require.NoError(t, err)
	assert.ElementsMatch(t, keys, keepAll)
}

// TestRenderValuesYAMLTrimsUnselectedCapabilityKeys：T6 的落盘口径（键裁剪函数），
// keep 全集时逐字节等于源文件。
func TestRenderValuesYAMLTrimsUnselectedCapabilityKeys(t *testing.T) {
	src := readRepoFile(t, "deploy/helm/values.yaml")
	keys, _, err := SectionBlocks(src)
	require.NoError(t, err)
	all, err := RenderValuesYAML(src, keys)
	require.NoError(t, err)
	assert.Equal(t, string(src), string(all))

	minimal, err := ParseCapabilitySet("minimal", "", "")
	require.NoError(t, err)
	keep, err := ValuesSections(src, minimal)
	require.NoError(t, err)
	out, err := RenderValuesYAML(src, keep)
	require.NoError(t, err)
	assert.NotContains(t, string(out), "\naudit:\n")
	assert.NotContains(t, string(out), "\nstorage:\n")
	assert.Contains(t, string(out), "\nauth:\n")
	assert.Contains(t, string(out), "\nimage:\n")
}
