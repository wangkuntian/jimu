package authmodule

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestModuleDescriptor 描述符：软依赖 captcha/breach（缺失即降级），
// 自有 login_histories/password_histories 表。
func TestModuleDescriptor(t *testing.T) {
	d := Descriptor
	assert.Equal(t, []string{"captcha", "breach"}, d.SoftRequires)
	assert.ElementsMatch(t, []string{"login_histories", "password_histories"}, d.Owns)
}
