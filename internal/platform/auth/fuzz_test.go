package auth

import (
	"testing"
)

// FuzzJWTParse 保证任意字符串下 JWT 解析不 panic，只返回错误。
func FuzzJWTParse(f *testing.F) {
	j := New("fuzz-test-secret", "jimu", 15, 7)
	if tok, err := j.GenerateAccess(1, "sess-1"); err == nil {
		f.Add(tok)
	}
	f.Add("not.a.jwt")
	f.Add("")
	f.Fuzz(func(t *testing.T, token string) {
		_, _ = j.Parse(token, TokenTypeAccess)
	})
}
