package authmodule

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestModuleDescriptor 描述符：硬依赖 user/access，软依赖 tenant/mfa/captcha/breach
// （缺失即降级：开通式注册关闭、二次验证跳过、验证码/泄露检查跳过），
// 自有 login_histories/password_histories 表。
func TestModuleDescriptor(t *testing.T) {
	d := Descriptor
	assert.Equal(t, []string{"user", "access"}, d.Requires)
	assert.Equal(t, []string{"tenant", "mfa", "captcha", "breach"}, d.SoftRequires)
	assert.ElementsMatch(t, []string{"login_histories", "password_histories"}, d.Owns)
}
