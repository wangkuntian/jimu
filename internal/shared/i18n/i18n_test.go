package i18n

import "testing"

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
