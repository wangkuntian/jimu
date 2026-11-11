package searchcore

// CJKFallbackScore 命中标题时的相关度分值（标题命中优先于正文命中）。
const CJKFallbackScore = 2

// ContainsCJK 判断查询串是否含 CJK 字符。
func ContainsCJK(s string) bool {
	for _, r := range s {
		if isCJKRune(r) {
			return true
		}
	}
	return false
}

func isCJKRune(r rune) bool {
	switch {
	case r >= 0x4E00 && r <= 0x9FFF:
		return true
	case r >= 0x3400 && r <= 0x4DBF:
		return true
	case r >= 0xF900 && r <= 0xFAFF:
		return true
	case r >= 0x3040 && r <= 0x30FF:
		return true
	case r >= 0xAC00 && r <= 0xD7AF:
		return true
	case r >= 0x20000 && r <= 0x2FA1F:
		return true
	default:
		return false
	}
}
