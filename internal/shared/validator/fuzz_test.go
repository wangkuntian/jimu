package validator

import (
	"testing"
)

// FuzzValidateRules 保证任意字符串下自定义校验规则不 panic（v.Var 只返回校验结果）。
func FuzzValidateRules(f *testing.F) {
	f.Add("13812345678")
	f.Add("Abc12345!")
	f.Add("")
	v := Validate()
	if v == nil {
		f.Skip("validator engine unavailable")
	}
	f.Fuzz(func(t *testing.T, s string) {
		for _, rule := range []string{"mobile", "password", "idcard", "username"} {
			_ = v.Var(s, rule)
		}
	})
}
