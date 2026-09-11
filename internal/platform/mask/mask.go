// Package mask 提供常用敏感信息的脱敏工具，用于日志与对外响应，
// 避免手机号、邮箱、证件号等 PII 明文落盘或外泄。
//
// 约定：脱敏只用于展示与日志，不改变存储；需要可逆保护的数据用字段级加密
// （platform/db 的 encryption tag + security.encryption_key）。
package mask

import "strings"

// 脱敏占位符
const redacted = "***"

// Phone 手机号脱敏：保留前 3 位与后 4 位（138***8888）。
// 长度不足时按比例保留，避免越界。
func Phone(s string) string {
	return keepMiddle(s, 3, 4, 7)
}

// Email 邮箱脱敏：保留首字符与完整域名（a***@example.com）。
func Email(s string) string {
	at := strings.LastIndex(s, "@")
	if at <= 0 {
		// 非邮箱格式：整体按普通字符串处理
		return keepMiddle(s, 1, 0, 3)
	}
	local, domain := s[:at], s[at:]
	if len(local) <= 1 {
		return redacted + domain
	}
	return local[:1] + redacted + domain
}

// IDCard 身份证号脱敏：保留前 6 位与后 4 位（110101***1234）。
func IDCard(s string) string {
	return keepMiddle(s, 6, 4, 11)
}

// BankCard 银行卡号脱敏：保留后 4 位（***1234）。
func BankCard(s string) string {
	if len(s) <= 4 {
		return redacted
	}
	return redacted + s[len(s)-4:]
}

// Name 姓名脱敏：保留姓氏（张*），两字姓名保留首字。
func Name(s string) string {
	r := []rune(s)
	if len(r) <= 1 {
		return redacted
	}
	return string(r[:1]) + strings.Repeat("*", len(r)-1)
}

// IP IP 地址脱敏：IPv4 保留前两段（192.168.*.*）。
func IP(s string) string {
	parts := strings.Split(s, ".")
	if len(parts) == 4 {
		return parts[0] + "." + parts[1] + ".*.*"
	}
	return s
}

// String 通用脱敏：保留前 prefix 位与后 suffix 位，中间用 *** 代替。
func String(s string, prefix, suffix int) string {
	return keepMiddle(s, prefix, suffix, 0)
}

// keepMiddle 保留前 prefix 与后 suffix 位；当长度不足以同时保留时整体替换。
// minKeep 为期望的最小保留长度（低于该长度视为过短，整体脱敏）。
func keepMiddle(s string, prefix, suffix, minKeep int) string {
	if s == "" {
		return ""
	}
	if minKeep > 0 && len(s) < minKeep {
		return redacted
	}
	if len(s) <= prefix+suffix {
		return redacted
	}
	if suffix == 0 {
		return s[:prefix] + redacted
	}
	return s[:prefix] + redacted + s[len(s)-suffix:]
}
