# v0.5.0 upgrade fixtures

这两个目录是 Jimu v0.5.0 生成的完整项目快照，用于 `jimu upgrade` 的发布门禁。
测试会把它们复制到临时目录再执行升级，不会修改这里的文件。

## 固定来源

| 模板 | module | 生成文件数 |
|---|---|---|
| `basic-api` | `example.com/upgrade-basic-api` | 22 |
| `auth-api` | `example.com/upgrade-auth-api` | 49 |

两个 manifest 都记录：

- `schema_version: 2`（v0.5.0 的只读 legacy 格式）
- `framework.version: v0.5.0`
- `framework.commit: 13cdc2a2519ed5f776e2c2a9c7e3fb36abfdedc0`
- `framework.dirty: false`

## 重新生成

fixture 由 v0.5.0 tag 源码按默认 `jimu new` 流程生成（包含 tidy），不在当前树上重新渲染。
重新生成时使用同一个 tag 和同一个 module；go.mod/go.sum 已由 v0.5.0 的 tidy 结果固定：

```bash
git worktree add --detach /tmp/jimu-v0.5.0 v0.5.0
cat >/tmp/gen-jimu-upgrade-fixture.go <<'EOF'
package main

import (
	"fmt"
	"os"

	"jimu/tools/generator"
)

func main() {
	res, err := generator.NewProject(generator.NewOptions{
		Dir: os.Args[2], Template: os.Args[1],
		Module: "example.com/upgrade-" + os.Args[1],
		NoSelfCheck: true, Force: true,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(res.FileCount)
}
EOF
cd /tmp/jimu-v0.5.0
go run -ldflags "-X jimu/internal/profiles.Version=v0.5.0" \
  /tmp/gen-jimu-upgrade-fixture.go basic-api \
  /path/to/tools/generator/testdata/upgrade/v0.5/basic-api
go run -ldflags "-X jimu/internal/profiles.Version=v0.5.0" \
  /tmp/gen-jimu-upgrade-fixture.go auth-api \
  /path/to/tools/generator/testdata/upgrade/v0.5/auth-api
```

`TestUpgradeFixturesMatchV050` 会校验 manifest digest、schema、provenance 和每个
`generated_files` 的哈希。手工修改 fixture 或重新生成时漏文件都会让该测试失败。
