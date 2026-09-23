package active

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestAssemblyDefaultsToFull 钉住提交态默认形态 = full：overlay 只在构建期替换选点文件
// （tools/profileoverlay），工作区里的 internal/profiles/active 恒指向 full，否则
// `go build ./cmd/server`、`go test ./...`、IDE 与 swagger 会与出货形态漂移。
//
// 这里必须比 Name 字面量：写成 full.Assembly() 的比较会变成恒真（选点包 import 的就是 full）。
func TestAssemblyDefaultsToFull(t *testing.T) {
	require.Equal(t, "full", Assembly().Name)
}
