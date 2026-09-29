package generator

import (
	"bytes"
	"go/scanner"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGeneratorTestHelpersDoNotInheritAmbientEnv 静态钉住「本包的测试 helper 起子进程时不得继承外界
// 环境」这条不变量 —— 它被违反过两次，而且**只在带 DB_*/Redis 的 tag 发布路径暴露**（ci-scaffold.yml
// 不注入这些键，所以 PR 路径一直是绿的）：
//
//	① `go run ./cmd/server` 继承 DB_* → 生成的服务器连上库、成功启动并常驻 → CombinedOutput 等到包级
//	   `-timeout 60m`（v0.3.0 首次发布的 release run 实测卡住 51 分 7 秒，整步 69 分 17 秒后失败）
//	② 生成项目的 `go test ./...` 继承 DB_* → 本仓复制过去的集成测试从「库不可达则跳过」变成「真跑」，
//	   重型矩阵并行生成十几个项目各自对同一个 jimu_test 库跑 goose 迁移 → 随机撞
//	   `Error 1060 Duplicate column name 'totp_secret'`，整片 TestGeneratedProjectTestTreeIsGreen 全红
//
// 修法是把这些子进程的 env 收口到 serverEnvBaseForTest / serverEnvForTest / closedDBEnvForTest
// （只透传 go 工具链变量 + PATH/HOME，应用配置键由调用方显式给出）。本用例扫描包内全部 *_test.go，
// 去掉注释后断言不出现 os.Environ()，让第三次只能改在明面上。
func TestGeneratorTestHelpersDoNotInheritAmbientEnv(t *testing.T) {
	files, err := filepath.Glob("*_test.go")
	require.NoError(t, err)
	require.NotEmpty(t, files, "扫不到 *_test.go：cwd 应为本包目录")

	// 用正则而不是字面量：stripGoComments 重建 token 文本时会插入空格（`os . Environ ( )`），
	// 字面量匹配会漏判；同时正则本身也不会把本文件里的字符串拼成能自匹配的字面量。
	needle := regexp.MustCompile(`os\s*\.\s*Environ\s*\(\s*\)`)
	for _, f := range files {
		src, err := os.ReadFile(f)
		require.NoError(t, err)
		assert.Nil(t, needle.FindStringIndex(stripGoComments(t, src)),
			"%s 起子进程不得继承外界环境（其应用配置键只在发布路径存在）：请用 serverEnvBaseForTest(cache) 或显式固定变量的构造器", f)
	}
}

// stripGoComments 用 go/scanner 去掉注释后重建源码，避免把注释里提到的 os.Environ() 误判为违规。
func stripGoComments(t *testing.T, src []byte) string {
	t.Helper()
	var buf bytes.Buffer
	fset := token.NewFileSet()
	file := fset.AddFile("", fset.Base(), len(src))
	var s scanner.Scanner
	s.Init(file, src, nil, scanner.ScanComments)
	for {
		_, tok, lit := s.Scan()
		if tok == token.EOF {
			break
		}
		if tok == token.COMMENT {
			continue
		}
		if lit != "" {
			buf.WriteString(lit)
		} else {
			buf.WriteString(tok.String())
		}
		buf.WriteByte(' ')
	}
	return buf.String()
}
