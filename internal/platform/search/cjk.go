package search

import "strings"

// cjkFallbackScore 命中标题时的相关度分值（标题命中优先于正文命中）
const cjkFallbackScore = 2

// containsCJK 判断查询串是否含 CJK 字符。
// 默认的 MySQL FULLTEXT 与 PostgreSQL tsvector 分词器按空白/词边界切词，
// 中文、日文、韩文没有词边界，因此这类查询需要退化为 LIKE 匹配才能命中。
func containsCJK(s string) bool {
	for _, r := range s {
		if isCJKRune(r) {
			return true
		}
	}
	return false
}

// isCJKRune 判断字符是否属于 CJK 相关区段（含中文、日文假名、韩文音节与扩展区）
func isCJKRune(r rune) bool {
	switch {
	case r >= 0x4E00 && r <= 0x9FFF: // CJK 统一表意文字
		return true
	case r >= 0x3400 && r <= 0x4DBF: // 扩展 A
		return true
	case r >= 0xF900 && r <= 0xFAFF: // 兼容表意文字
		return true
	case r >= 0x3040 && r <= 0x30FF: // 日文平假名/片假名
		return true
	case r >= 0xAC00 && r <= 0xD7AF: // 韩文音节
		return true
	case r >= 0x20000 && r <= 0x2FA1F: // 扩展 B-F
		return true
	default:
		return false
	}
}

// likeEscapeChar LIKE 的转义字符。用 '!' 而不是反斜杠：MySQL 字符串字面量里
// 反斜杠本身是转义字符，跨方言写起来易错；'!' 在两种数据库中都无需额外转义。
const likeEscapeChar = "!"

// escapeLike 转义 LIKE 通配符，配合 ESCAPE '!' 使用
func escapeLike(s string) string {
	replacer := strings.NewReplacer(
		likeEscapeChar, likeEscapeChar+likeEscapeChar,
		"%", likeEscapeChar+"%",
		"_", likeEscapeChar+"_",
	)
	return replacer.Replace(s)
}

// likePattern 生成 LIKE 模式（子串匹配）
func likePattern(query string) string {
	return "%" + escapeLike(query) + "%"
}
