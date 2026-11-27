package i18n

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCapabilityMessagesFormat(t *testing.T) {
	assert.Equal(t, "账号已被锁定，请 5 分钟后重试", Tf("account_locked", LangZH, 5))
	assert.Equal(t, "account locked, try again in 30 minutes", Tf("account_locked", LangEN, 30))
}
