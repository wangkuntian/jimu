// Package sqlpattern 提供跨数据库一致的 SQL LIKE 模式构造。
package sqlpattern

import "strings"

// EscapeChar LIKE 的转义字符。用 '!' 而不是反斜杠：MySQL 字符串字面量里
// 反斜杠本身是转义字符，跨方言写起来易错；'!' 在 MySQL/PostgreSQL/SQLite 中
// 都是普通字面量，无需额外转义。
const EscapeChar = "!"

// EscapeLike 转义 LIKE 通配符，配合 ESCAPE '!' 使用。
func EscapeLike(s string) string {
	replacer := strings.NewReplacer(
		EscapeChar, EscapeChar+EscapeChar,
		"%", EscapeChar+"%",
		"_", EscapeChar+"_",
	)
	return replacer.Replace(s)
}

// LikeContains 生成子串匹配的 LIKE 模式（调用方仍需显式写 ESCAPE '!'）。
func LikeContains(query string) string {
	return "%" + EscapeLike(query) + "%"
}
