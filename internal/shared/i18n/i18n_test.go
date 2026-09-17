package i18n

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTDefaultsToChinese(t *testing.T) {
	if got := T("internal_error"); got != "服务器内部错误" {
		t.Fatalf("got %q", got)
	}
	if got := T("no_such_key"); got != "no_such_key" {
		t.Fatalf("missing key should return key itself, got %q", got)
	}
}

func TestTfFormatsArgs(t *testing.T) {
	if got := Tf("account_locked", LangZH, 30); got != "账号已被锁定，请 30 分钟后重试" {
		t.Fatalf("got %q", got)
	}
	if got := Tf("account_locked", LangEN, 30); got != "account locked, try again in 30 minutes" {
		t.Fatalf("got %q", got)
	}
}

func TestParseAcceptLanguage(t *testing.T) {
	tests := []struct {
		header string
		want   string
	}{
		{"", LangZH},
		{"en-US,en;q=0.9", LangEN},
		{"zh-CN,zh;q=0.8,en;q=0.7", LangZH},
		{"fr-FR,fr;q=0.9", LangZH}, // 不支持的语言回退默认
	}
	for _, tt := range tests {
		if got := ParseAcceptLanguage(tt.header); got != tt.want {
			t.Fatalf("ParseAcceptLanguage(%q) = %q, want %q", tt.header, got, tt.want)
		}
	}
}

func TestTfAndDefaultLanguage(t *testing.T) {
	// 默认语言为中文；Tf 在提供参数时格式化
	SetDefaultLang(LangZH)
	assert.Equal(t, "用户名或密码错误", T("invalid_credentials"))
	assert.Equal(t, "账号已被锁定，请 5 分钟后重试", Tf("account_locked", LangZH, 5))

	// 切换默认语言后 T 跟随默认值，显式传参仍以参数为准
	SetDefaultLang(LangEN)
	assert.Equal(t, "invalid username or password", T("invalid_credentials"))
	assert.Equal(t, "用户名或密码错误", T("invalid_credentials", LangZH))
	SetDefaultLang(LangZH)
}

func TestRegisterMessagesAndHas(t *testing.T) {
	assert.False(t, Has("custom_key", LangZH))

	RegisterMessages(LangZH, map[string]string{"custom_key": "自定义"})
	RegisterMessages("fr", map[string]string{"custom_key": "personnalisé"})
	t.Cleanup(func() {
		// 清理注册的语言，避免影响其他用例
		mu.Lock()
		delete(messages, "fr")
		delete(messages[LangZH], "custom_key")
		mu.Unlock()
	})

	assert.True(t, Has("custom_key", LangZH))
	assert.True(t, Has("custom_key", "fr"))
	assert.Equal(t, "自定义", T("custom_key", LangZH))
	assert.Equal(t, "personnalisé", T("custom_key", "fr"))
	// 显式指定未登记的语言时不做兜底（调用方应先经 ParseAcceptLanguage 归一）
	assert.Equal(t, "custom_key", T("custom_key", "de"))
}

func TestParseAcceptLanguageFallsBack(t *testing.T) {
	assert.Equal(t, LangEN, ParseAcceptLanguage("en-US,en;q=0.9"))
	assert.Equal(t, LangZH, ParseAcceptLanguage("zh-CN,zh;q=0.9"))
	// 未支持的语言回退默认语言
	SetDefaultLang(LangZH)
	assert.Equal(t, LangZH, ParseAcceptLanguage("de-DE,de;q=0.9"))
	assert.Equal(t, LangZH, ParseAcceptLanguage(""))
	// 多语言候选里取第一个受支持的
	assert.Equal(t, LangEN, ParseAcceptLanguage("de-DE,en;q=0.8"))
}

func TestShippedValidationMessagesFormat(t *testing.T) {
	// 校验类文案含占位符，Tf 应能正常格式化（防止 %s 数量与调用方不一致）
	assert.Equal(t, "username 不能为空", Tf("validation_required", LangZH, "username"))
	assert.Equal(t, "password must be 8-32 characters with letters and numbers", Tf("validation_password", LangEN, "password"))
}
