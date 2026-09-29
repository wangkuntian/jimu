package mask

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPhone(t *testing.T) {
	assert.Equal(t, "138***8888", Phone("13812348888"))
	assert.Equal(t, "***", Phone("123456")) // 过短整体脱敏
	assert.Equal(t, "", Phone(""))
}

func TestEmail(t *testing.T) {
	assert.Equal(t, "a***@example.com", Email("alice@example.com"))
	assert.Equal(t, "***@example.com", Email("a@example.com"))
	assert.Equal(t, "n***", Email("not-an-email")) // 无 @ 时按普通字符串保留首字符
}

func TestIDCardAndBankCard(t *testing.T) {
	assert.Equal(t, "110101***1234", IDCard("110101199001011234"))
	assert.Equal(t, "***", IDCard("12345"))
	assert.Equal(t, "***1234", BankCard("6222021234561234"))
	assert.Equal(t, "***", BankCard("123"))
}

func TestNameAndIP(t *testing.T) {
	assert.Equal(t, "张*", Name("张三"))
	assert.Equal(t, "欧**", Name("欧阳锋"))
	assert.Equal(t, "***", Name("张"))
	assert.Equal(t, "192.168.*.*", IP("192.168.1.10"))
	assert.Equal(t, "::1", IP("::1"))
}

func TestStringKeepsEdges(t *testing.T) {
	assert.Equal(t, "abc***xyz", String("abcdefxyz", 3, 3))
	assert.Equal(t, "ab***", String("abcdef", 2, 0))
	assert.Equal(t, "***", String("ab", 3, 3))
}

func TestRedactByKeyMatchesNormalizedNames(t *testing.T) {
	// 凭证类整体替换
	for _, key := range []string{"password", "Access-Token", "api_key", "APIKey", "client.secret", "TotpSecret"} {
		got, ok := RedactByKey(key, "supersecret")
		assert.True(t, ok, key)
		assert.Equal(t, "***", got, key)
	}

	// PII 部分保留
	got, ok := RedactByKey("email", "alice@example.com")
	assert.True(t, ok)
	assert.Equal(t, "a***@example.com", got)

	got, ok = RedactByKey("phone_number", "13812348888")
	assert.True(t, ok)
	assert.Equal(t, "138***8888", got)
}

func TestRedactByKeyLeavesUnknownAndEmpty(t *testing.T) {
	got, ok := RedactByKey("username", "alice")
	assert.False(t, ok)
	assert.Equal(t, "alice", got)

	// 精确匹配，不做子串匹配
	got, ok = RedactByKey("token_count", "42")
	assert.False(t, ok)
	assert.Equal(t, "42", got)

	got, ok = RedactByKey("password", "")
	assert.False(t, ok)
	assert.Equal(t, "", got)
}

func TestMapMasksNestedSensitiveFields(t *testing.T) {
	in := map[string]any{
		"username": "alice",
		"email":    "alice@example.com",
		"nested": map[string]any{
			"api_key": "k-123",
			"note":    "keep",
		},
		"items": []any{map[string]any{"phone": "13812348888"}},
	}
	out := Map(in)

	assert.Equal(t, "alice", out["username"])
	assert.Equal(t, "a***@example.com", out["email"])
	nested := out["nested"].(map[string]any)
	assert.Equal(t, "***", nested["api_key"])
	assert.Equal(t, "keep", nested["note"])
	items := out["items"].([]any)
	assert.Equal(t, "138***8888", items[0].(map[string]any)["phone"])

	// 不修改入参
	assert.Equal(t, "alice@example.com", in["email"])
}

func TestMapNil(t *testing.T) {
	assert.Nil(t, Map(nil))
}
