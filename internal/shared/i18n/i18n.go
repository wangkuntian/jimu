// Package i18n 提供简单的国际化支持
package i18n

import (
	"fmt"
	"strings"
	"sync"
)

// 支持的语言
const (
	LangEN = "en"
	LangZH = "zh"
)

// 默认语言
var defaultLang = LangZH

// 翻译存储
var (
	mu        sync.RWMutex
	messages  = map[string]map[string]string{
		LangZH: {
			"invalid_credentials":       "用户名或密码错误",
			"user_not_found":           "用户不存在",
			"user_exists":              "用户名已存在",
			"account_locked":           "账号已被锁定，请 %d 分钟后重试",
			"too_many_requests":        "请求过于频繁，请稍后再试",
			"internal_error":           "服务器内部错误",
			"invalid_param":            "参数错误",
			"not_found":                "资源不存在",
			"forbidden":                "无权限访问",
			"unauthorized":             "未认证",
			"conflict":                 "资源冲突",
			"timeout":                  "请求超时",
			"file_too_large":           "文件过大",
			"file_type_not_allowed":    "不支持的文件类型",
			"validation_failed":        "校验失败",
			"rate_limit_exceeded":      "超出限流阈值",
		},
		LangEN: {
			"invalid_credentials":       "invalid username or password",
			"user_not_found":           "user not found",
			"user_exists":              "username already exists",
			"account_locked":           "account locked, try again in %d minutes",
			"too_many_requests":        "too many requests, please try later",
			"internal_error":           "internal server error",
			"invalid_param":            "invalid parameter",
			"not_found":                "resource not found",
			"forbidden":                "access denied",
			"unauthorized":             "unauthorized",
			"conflict":                 "resource conflict",
			"timeout":                  "request timeout",
			"file_too_large":           "file too large",
			"file_type_not_allowed":    "file type not allowed",
			"validation_failed":        "validation failed",
			"rate_limit_exceeded":      "rate limit exceeded",
		},
	}
)

// SetDefaultLang 设置默认语言
func SetDefaultLang(lang string) {
	mu.Lock()
	defer mu.Unlock()
	defaultLang = lang
}

// T 翻译指定 key
func T(key string, lang ...string) string {
	mu.RLock()
	defer mu.RUnlock()

	l := defaultLang
	if len(lang) > 0 && lang[0] != "" {
		l = lang[0]
	}

	if m, ok := messages[l]; ok {
		if msg, ok := m[key]; ok {
			return msg
		}
	}
	return key
}

// Tf 翻译并格式化
func Tf(key string, lang string, args ...interface{}) string {
	msg := T(key, lang)
	if len(args) > 0 {
		return fmt.Sprintf(msg, args...)
	}
	return msg
}

// ParseAcceptLanguage 从 Accept-Language header 解析首选语言
func ParseAcceptLanguage(header string) string {
	if header == "" {
		return defaultLang
	}
	// 简单解析：取第一个语言标签
	parts := strings.Split(header, ",")
	for _, part := range parts {
		lang := strings.TrimSpace(strings.Split(part, ";")[0])
		if _, ok := messages[lang]; ok {
			return lang
		}
	}
	return defaultLang
}

// RegisterMessages 注册自定义翻译
func RegisterMessages(lang string, m map[string]string) {
	mu.Lock()
	defer mu.Unlock()
	if existing, ok := messages[lang]; ok {
		for k, v := range m {
			existing[k] = v
		}
	} else {
		messages[lang] = m
	}
}
