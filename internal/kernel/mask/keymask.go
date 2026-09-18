package mask

import "strings"

// redactFunc 按字段名命中后的脱敏函数
type redactFunc func(string) string

// 按字段名精确匹配（大小写与分隔符不敏感），避免 substring 误伤（如 token_count）。
var sensitiveKeys = map[string]redactFunc{
	// 凭证类：整体替换，绝不落盘
	"password":      fullRedact,
	"passwd":        fullRedact,
	"pwd":           fullRedact,
	"token":         fullRedact,
	"accesstoken":   fullRedact,
	"refreshtoken":  fullRedact,
	"idtoken":       fullRedact,
	"secret":        fullRedact,
	"clientsecret":  fullRedact,
	"apikey":        fullRedact,
	"authorization": fullRedact,
	"credential":    fullRedact,
	"credentials":   fullRedact,
	"cookie":        fullRedact,
	"setcookie":     fullRedact,
	"privatekey":    fullRedact,
	"sessionid":     fullRedact,
	"signature":     fullRedact,
	// PII：部分保留，便于排查
	"email":        Email,
	"mail":         Email,
	"emailaddress": Email,
	"phone":        Phone,
	"mobile":       Phone,
	"phonenumber":  Phone,
	"mobilenumber": Phone,
	"tel":          Phone,
	"idcard":       IDCard,
	"idnumber":     IDCard,
	"idno":         IDCard,
	"identitycard": IDCard,
	"nationalid":   IDCard,
	"bankcard":     BankCard,
	"cardno":       BankCard,
	"cardnumber":   BankCard,
	"creditcard":   BankCard,
	"passwordhash": fullRedact,
	"totpsecret":   fullRedact,
}

func fullRedact(string) string { return redacted }

// RedactByKey 按字段名脱敏：命中返回脱敏值与 true，未命中原样返回 false。
// 字段名匹配忽略大小写与 `_`/`-`/`.` 分隔符（access_token / accessToken 等价）。
func RedactByKey(key, value string) (string, bool) {
	if value == "" {
		return value, false
	}
	fn, ok := sensitiveKeys[normalizeKey(key)]
	if !ok {
		return value, false
	}
	return fn(value), true
}

// Map 递归脱敏 map 中命中敏感字段名的字符串值，用于日志与事件载荷。
// 返回新 map，不修改入参。
func Map(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		switch val := v.(type) {
		case string:
			masked, _ := RedactByKey(k, val)
			out[k] = masked
		case map[string]any:
			out[k] = Map(val)
		case []any:
			out[k] = maskSlice(val)
		default:
			out[k] = v
		}
	}
	return out
}

func maskSlice(in []any) []any {
	out := make([]any, len(in))
	for i, v := range in {
		if m, ok := v.(map[string]any); ok {
			out[i] = Map(m)
			continue
		}
		out[i] = v
	}
	return out
}

// normalizeKey 归一化字段名：小写并去掉分隔符
func normalizeKey(key string) string {
	var b strings.Builder
	b.Grow(len(key))
	for _, r := range key {
		switch r {
		case '_', '-', '.', ' ':
			continue
		}
		if r >= 'A' && r <= 'Z' {
			r += 'a' - 'A'
		}
		b.WriteRune(r)
	}
	return b.String()
}
