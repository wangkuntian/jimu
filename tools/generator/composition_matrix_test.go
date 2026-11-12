package generator

import (
	"os"
	"path/filepath"
	"testing"

	"jimu/internal/capability"
	"jimu/internal/contract"
	"jimu/tools/generator/manifest"
	"jimu/tools/generator/workspace"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// compositionMatrixCases 是 full 档的固定、代表性 pairwise 清单，不是全幂集。
//
// 每个用例最多请求三个能力：两个代表性能力加必要的受保护中间件提供方
// （auth 或 apikey）。用例覆盖端口提供/消费、驱动、迁移与中间件交互；
// 每个生成项目只做 build/vet，不承诺组合级 E2E。
//
// 预算：最多 compositionMatrixMaxCases 个生成项目、compositionMatrixShards 个
// CI shard。新增能力或驱动时，按“与现有能力的一个真实交互”补一行，不要改成
// 遍历所有组合。失败时子测试名和日志都带 composition / capability / driver。
const (
	compositionMatrixMaxCases = 10
	compositionMatrixShards   = 2
)

type compositionMatrixCase struct {
	name     string
	with     string
	resolved []string
	drivers  map[string][]string
}

var compositionMatrixCases = []compositionMatrixCase{
	{
		name:     "auth-passkey",
		with:     "passkey",
		resolved: []string{"user", "access", "auth", "passkey"},
		drivers:  map[string][]string{"auth": {"redis"}},
	},
	{
		name:     "auth-grpc",
		with:     "auth,grpc",
		resolved: []string{"user", "access", "auth", "grpc"},
		drivers:  map[string][]string{"auth": {"redis"}},
	},
	{
		name:     "auth-console-ws",
		with:     "auth,console,ws",
		resolved: []string{"user", "access", "auth", "console", "ws"},
		drivers:  map[string][]string{"auth": {"redis"}},
	},
	{
		name:     "apikey-storage-s3-uploadsec",
		with:     "apikey,storage:s3,uploadsec",
		resolved: []string{"apikey", "uploadsec", "storage"},
		drivers:  map[string][]string{"storage": {"s3"}},
	},
	{
		name:     "apikey-queue-kafka-outbox",
		with:     "apikey,queue:kafka,outbox",
		resolved: []string{"apikey", "queue", "outbox"},
		drivers:  map[string][]string{"queue": {"kafka"}},
	},
	{
		name:     "apikey-queue-rabbitmq-outbox",
		with:     "apikey,queue:rabbitmq,outbox",
		resolved: []string{"apikey", "queue", "outbox"},
		drivers:  map[string][]string{"queue": {"rabbitmq"}},
	},
	{
		name:     "apikey-dataops-excel-search",
		with:     "apikey,dataops:excel,search",
		resolved: []string{"apikey", "dataops", "search"},
		drivers:  map[string][]string{"dataops": {"excel"}, "search": {"mysql"}},
	},
	{
		name:     "auth-captcha-notification",
		with:     "auth,captcha,notification",
		resolved: []string{"user", "access", "auth", "captcha", "notification"},
		drivers:  map[string][]string{"auth": {"redis"}},
	},
	{
		name:     "oauth-mfa",
		with:     "oauth,mfa",
		resolved: []string{"user", "access", "mfa", "auth", "oauth"},
		drivers:  map[string][]string{"auth": {"redis"}},
	},
	{
		name:     "auth-breach-encryption",
		with:     "auth,breach,encryption",
		resolved: []string{"user", "access", "auth", "breach", "encryption"},
		drivers:  map[string][]string{"auth": {"redis"}},
	},
}

// TestRepresentativeCompositionMatrix 是 full 档的组合回归网：每个固定用例都生成
// 真实项目，校验 manifest 的 resolved/drivers 事实，再执行 go build/vet。失败日志
// 带 composition、选择表达式、resolved 与 drivers，能直接定位到能力/驱动。
func TestRepresentativeCompositionMatrix(t *testing.T) {
	requireHeavyMatrix(t)

	indices, err := scaffoldShardIndices(os.Getenv(scaffoldShardEnv), len(compositionMatrixCases))
	require.NoError(t, err)

	cache := newTestGoCache(t)
	sem := make(chan struct{}, heavyBuildConcurrency)
	for _, index := range indices {
		index := index
		tc := compositionMatrixCases[index]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			sem <- struct{}{}
			defer func() { <-sem }()

			t.Logf("composition=%s with=%s resolved=%v drivers=%v", tc.name, tc.with, tc.resolved, tc.drivers)
			dir := filepath.Join(t.TempDir(), "proj")
			_, err := newProjectForTest(t, NewOptions{
				Dir: dir, With: tc.with, Module: "example.com/proj", NoTidy: true,
			})
			require.NoError(t, err, "composition=%s with=%s must generate", tc.name, tc.with)

			doc, err := workspace.LoadProjectManifest(dir)
			require.NoError(t, err)
			require.NotNil(t, doc.Composition)
			assert.Equal(t, manifest.CompositionSupportExperimental, doc.Composition.Support)
			assert.ElementsMatch(t, tc.resolved, doc.Composition.Resolved,
				"composition=%s with=%s resolved", tc.name, tc.with)
			assert.Len(t, doc.Selection.Drivers, len(tc.drivers),
				"composition=%s with=%s drivers", tc.name, tc.with)
			for name, want := range tc.drivers {
				assert.Equal(t, want, doc.Selection.Drivers[name],
					"composition=%s capability=%s driver", tc.name, name)
			}

			out, err := runGoInProjectOutput(t, dir, cache, "build", "./...")
			require.NoError(t, err, "composition=%s with=%s go build failed:\n%s", tc.name, tc.with, out)
			out, err = runGoInProjectOutput(t, dir, cache, "vet", "./...")
			require.NoError(t, err, "composition=%s with=%s go vet failed:\n%s", tc.name, tc.with, out)
		})
	}
}

// TestCompositionMatrixBudget 钉住“不遍历全幂集”的预算：清单长度有上限，
// 每个用例最多三个请求能力，名称唯一。
func TestCompositionMatrixBudget(t *testing.T) {
	require.LessOrEqual(t, len(compositionMatrixCases), compositionMatrixMaxCases,
		"pairwise 清单超出预算；请合并代表性用例，不要遍历全幂集")
	require.GreaterOrEqual(t, compositionMatrixShards, 1)

	seen := make(map[string]bool, len(compositionMatrixCases))
	for _, tc := range compositionMatrixCases {
		require.NotEmpty(t, tc.name)
		require.False(t, seen[tc.name], "duplicate composition case %q", tc.name)
		seen[tc.name] = true
		require.NotEmpty(t, tc.resolved, "composition=%s resolved budget", tc.name)

		names, overrides, err := parseWith(tc.with)
		require.NoError(t, err, "composition=%s with=%s", tc.name, tc.with)
		require.LessOrEqual(t, len(names), 3,
			"composition=%s 必须保持 pairwise + 中间件提供方，不允许堆叠全组合", tc.name)
		for name := range overrides {
			require.Contains(t, names, name, "composition=%s driver overwrite for unselected capability", tc.name)
		}
	}
}

// negativeCompositionCase 是固定 catalog 上的无效组合。它们必须在写目标目录前失败，
// 且错误码/能力名能定位到具体选择。
type negativeCompositionCase struct {
	name           string
	with           string
	wantCode       capability.DiagnosticCode
	wantCapability string
	wantMessage    string
}

var negativeCompositionCases = []negativeCompositionCase{
	{
		name:        "protected-routes-without-provider",
		with:        "user",
		wantCode:    capability.CodeMiddlewareProviderMissing,
		wantMessage: "no capability provides protected middleware",
	},
	{
		name:           "unknown-driver",
		with:           "queue:ghost",
		wantCode:       capability.CodeUnknownDriver,
		wantCapability: "queue",
		wantMessage:    `unknown driver "ghost"`,
	},
	{
		name:           "duplicate-capability",
		with:           "queue,queue",
		wantCode:       capability.CodeDuplicateCapability,
		wantCapability: "queue",
		wantMessage:    "more than once",
	},
	{
		name:           "unknown-capability",
		with:           "ghost",
		wantCode:       capability.CodeUnknownCapability,
		wantCapability: "ghost",
		wantMessage:    `unknown capability "ghost"`,
	},
}

// TestCompositionNegativeMatrix 覆盖 full 档的冲突/负例：固定 catalog 上的无效选择、
// 同一能力选两个驱动，以及 resolver 的 conflict 诊断路径。所有失败都不得写目标目录。
func TestCompositionNegativeMatrix(t *testing.T) {
	for _, tc := range negativeCompositionCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "proj")
			_, err := newProjectForTest(t, NewOptions{
				Dir: dir, With: tc.with, Module: "example.com/proj", NoTidy: true,
			})
			require.Error(t, err, "composition=%s with=%s must fail", tc.name, tc.with)

			var compositionErr *capability.Error
			require.ErrorAs(t, err, &compositionErr, "failure must carry structured diagnostics: %v", err)
			diag := compositionDiagnostic(compositionErr.Diagnostics, tc.wantCode)
			require.NotNil(t, diag, "composition=%s diagnostics=%#v, want code %q", tc.name, compositionErr.Diagnostics, tc.wantCode)
			assert.Equal(t, tc.wantCapability, diag.Capability, "composition=%s capability", tc.name)
			assert.Contains(t, diag.Message, tc.wantMessage, "composition=%s message", tc.name)

			_, statErr := os.Stat(dir)
			assert.True(t, os.IsNotExist(statErr), "composition=%s must fail before writing the target", tc.name)
		})
	}

	t.Run("same-capability-two-drivers", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "proj")
		_, err := newProjectForTest(t, NewOptions{
			Dir: dir, With: "queue:redis,queue:kafka", Module: "example.com/proj", NoTidy: true,
		})
		require.Error(t, err)
		require.ErrorContains(t, err, `capability "queue" selects two drivers`)
		_, statErr := os.Stat(dir)
		assert.True(t, os.IsNotExist(statErr), "driver conflict must fail before writing the target")
	})

	t.Run("synthetic-conflict", func(t *testing.T) {
		// 固定 catalog 当前没有声明 Descriptor.Conflicts 的能力对；这里用一对合成
		// 描述符钉住 conflict 诊断路径，避免 full 档只覆盖“非法输入”而不覆盖声明冲突。
		result := capability.Compose([]contract.Descriptor{
			{Name: "alpha", Conflicts: []string{"beta"}},
			{Name: "beta"},
		}, capability.ComposeRequest{Capabilities: []string{"alpha", "beta"}})
		diag := compositionDiagnostic(result.Diagnostics, capability.CodeConflict)
		require.NotNil(t, diag)
		assert.Equal(t, "alpha", diag.Capability)
		assert.Contains(t, diag.Message, `conflicts with "beta"`)
	})
}

func compositionDiagnostic(values []capability.Diagnostic, code capability.DiagnosticCode) *capability.Diagnostic {
	for index := range values {
		if values[index].Code == code {
			return &values[index]
		}
	}
	return nil
}
