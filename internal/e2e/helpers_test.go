package e2e

import (
	"os"
	"path/filepath"
	"testing"

	"jimu/internal/config"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

// testSections 用**真实** configs/app.yaml 构造内存 SectionDecoder。
//
// 为什么用真实配置而不是手写结构体：能力配置段由各能力自己声明（Descriptor.Configs），
// 手写会与真实段漂移；P2.6 T5 起 e2e 按形态装配，装配期解码的正是这份配置
// （app.LoadCapabilityConfigs），所以它必须与生产配置同源。
func testSections(t *testing.T) config.SectionDecoder {
	t.Helper()

	v := viper.New()
	v.SetConfigType("yaml")
	f, err := os.Open(filepath.Join("configs", "app.yaml"))
	require.NoError(t, err)
	defer func() { _ = f.Close() }()
	require.NoError(t, v.ReadConfig(f))

	// 唯一的覆盖：把本地存储目录指向临时目录 —— storage/local 会 MkdirAll(base_dir)，而 e2e
	// 必须留在仓库根（casbin 要按相对路径读 conf/rbac_model.conf），否则每次跑测都会在仓库根
	// 留下空的 storage/ 目录。其余保持真实 dev 配置（能力配置要过各自 Validate()，如 auth 要求
	// 限流值 > 0，不能为了测试把它改小/改 0）。
	v.Set("storage.base_dir", t.TempDir())
	return viperSectionDecoder{v: v}
}

// viperSectionDecoder 把 viper 适配为 config.SectionDecoder（与 internal/app/capconfig_test.go 同构）。
type viperSectionDecoder struct{ v *viper.Viper }

func (d viperSectionDecoder) UnmarshalKey(key string, rawVal any) error {
	return d.v.UnmarshalKey(key, rawVal)
}
