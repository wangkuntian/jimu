package profiles_test

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"jimu/internal/app"
	"jimu/internal/assembly"
	"jimu/internal/capabilities/catalog"
	userdomain "jimu/internal/capabilities/user/domain"
	"jimu/internal/config"
	"jimu/internal/contract"
	"jimu/internal/kernel/logger"
	"jimu/internal/profiles"
	"jimu/internal/profiles/enterprise"
	"jimu/internal/profiles/full"
	"jimu/internal/profiles/machine"
	"jimu/internal/profiles/minimal"
	"jimu/internal/profiles/saas"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// profileAssemblies 是设计 §2 的 5 个形态入口。
func profileAssemblies() []struct {
	name string
	a    assembly.Assembly
} {
	return []struct {
		name string
		a    assembly.Assembly
	}{
		{"full", full.Assembly()},
		{"minimal", minimal.Assembly()},
		{"saas", saas.Assembly()},
		{"enterprise", enterprise.Assembly()},
		{"machine", machine.Assembly()},
	}
}

// TestProfilePortFlow 每个形态的端口流向护栏（Task 3 评审：护栏此前只覆盖 full）。
//
// ValidatePortFlow 把「读取时尚未提供」一律视为违规，这对裁剪形态会误报：形态刻意排除
// 的能力所提供的端口（如 minimal 的 user 读 tenant/outbox）本就缺席，读取它们正是设计内的
// 软依赖降级。因此这里先用 ProbeAssembly 观测 full 提供的端口全集，把形态缺席的端口以
// **空值**前置注册（桩只登记端口名，消费方取回的值仍是 nil，降级行为不变），使护栏只对
// 形态**内部**真实的端口边报错——把 access 排到 user 之后、或把 grpc 排到 user 之前都会
// 让本用例失败。
func TestProfilePortFlow(t *testing.T) {
	universe := providedPorts(t, full.Assembly())
	for _, p := range profileAssemblies() {
		t.Run(p.name, func(t *testing.T) {
			require.NoError(t, assembly.ValidatePortFlow(withAbsentPortStubs(t, p.a, universe)))
		})
	}
}

// TestProfileNonCatalogEntriesAreUngated 每个形态里非 catalog 条目都必须标记 Ungated
// （Task 3 评审 Finding 1 的复发风险：形态若把 encryption/storage 当受门控条目，
// capabilities.enabled 的非空子集会把它们静默裁剪，字段加密/存储端口随之消失）；
// catalog 条目则必须受门控。
func TestProfileNonCatalogEntriesAreUngated(t *testing.T) {
	inCatalog := map[string]bool{}
	for _, n := range catalog.Names() {
		inCatalog[n] = true
	}
	for _, p := range profileAssemblies() {
		t.Run(p.name, func(t *testing.T) {
			for _, c := range p.a.Capabilities {
				if inCatalog[c.Descriptor.Name] {
					require.False(t, c.Ungated, "catalog 能力 %q 不得标记 Ungated", c.Descriptor.Name)
					continue
				}
				require.True(t, c.Ungated,
					"非 catalog 条目 %q 必须标记 Ungated，否则会被 capabilities.enabled 静默裁剪", c.Descriptor.Name)
			}
		})
	}
}

// withAbsentPortStubs 前置一个只登记「本形态缺席端口」的桩能力，见 TestProfilePortFlow。
func withAbsentPortStubs(t *testing.T, a assembly.Assembly, universe map[string]bool) assembly.Assembly {
	t.Helper()

	present := map[string]bool{}
	for name := range providedPorts(t, a) {
		present[name] = true
	}
	absent := make([]string, 0, len(universe))
	for name := range universe {
		if !present[name] {
			absent = append(absent, name)
		}
	}
	if len(absent) == 0 {
		return a
	}
	// 端口名排序：试运行结果与 map 迭代顺序无关。
	sort.Strings(absent)

	stub := assembly.Capability{
		Descriptor: contract.Descriptor{Name: "absent-ports-probe"},
		Wire: func(ctx *assembly.Context) (contract.Module, error) {
			for _, name := range absent {
				if err := ctx.Provide(name, nil); err != nil {
					return nil, err
				}
			}
			return nil, nil
		},
	}
	stubbed := a
	stubbed.Capabilities = append([]assembly.Capability{stub}, a.Capabilities...)
	return stubbed
}

// providedPorts 返回形态解析集里被任一能力 Provide 的端口名集合。
func providedPorts(t *testing.T, a assembly.Assembly) map[string]bool {
	t.Helper()

	res, err := assembly.ProbeAssembly(a, nil)
	require.NoError(t, err)

	out := map[string]bool{}
	for _, ports := range res.Provided {
		for _, name := range ports {
			out[name] = true
		}
	}
	return out
}

// newProfilesSqliteDB 打开独立的内存 sqlite（命名共享缓存的理由同 internal/app/seed_test.go：
// 普通 :memory: 每连接一个库，事务与查询可能落在不同连接上看不到表）。
func newProfilesSqliteDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:profiles_seed_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

// TestStructuralSeedSkipsWithNoAdminPassword 钉住启动行为不变：ADMIN_PASSWORD 未设置时
// StructuralSeed 必须正常返回 nil（容器启动早于 CLI 迁移，表可能还不存在），且只有库中
// 确实一个账号都没有时才补 error 级自检提示（app.WarnIfNoAccounts）。
func TestStructuralSeedSkipsWithNoAdminPassword(t *testing.T) {
	t.Setenv("ADMIN_PASSWORD", "")

	cases := []struct {
		name       string
		migrate    bool
		wantNoAcct bool
	}{
		{name: "empty database warns and returns nil", migrate: true, wantNoAcct: true},
		{name: "unmigrated database stays quiet and returns nil", migrate: false, wantNoAcct: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := newProfilesSqliteDB(t)
			if tc.migrate {
				require.NoError(t, db.AutoMigrate(&userdomain.User{}))
			}

			logPath := filepath.Join(t.TempDir(), "server.log")
			log := logger.New(config.LogConfig{Output: logPath})
			ctx, _, err := assembly.WireFor(
				&app.Container{Config: &config.Config{}, DB: db, Logger: log},
				nil, nil, assembly.Assembly{Name: "structural-seed-test"}, nil,
			)
			require.NoError(t, err)

			require.NoError(t, profiles.StructuralSeed(ctx), "跳过路径必须返回 nil，服务启动不得失败")

			out, err := os.ReadFile(logPath)
			require.NoError(t, err)
			if tc.wantNoAcct {
				require.Contains(t, string(out), "no account exists")
			} else {
				require.NotContains(t, string(out), "no account exists")
			}
		})
	}
}
