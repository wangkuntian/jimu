package middleware

import (
	"bytes"
	"strings"

	"github.com/gin-gonic/gin"
)

// responseBodyWriter 包装 gin.ResponseWriter 以捕获响应体
type responseBodyWriter struct {
	gin.ResponseWriter
	body      *bytes.Buffer
	maxSize   int
	truncated bool
}

func newResponseBodyWriter(w gin.ResponseWriter, maxSize int) *responseBodyWriter {
	return &responseBodyWriter{
		ResponseWriter: w,
		body:           &bytes.Buffer{},
		maxSize:        maxSize,
	}
}

func (w *responseBodyWriter) Write(b []byte) (int, error) {
	// 只捕获到限制大小，超出后标记截断
	if w.body.Len() < w.maxSize {
		remaining := w.maxSize - w.body.Len()
		if len(b) <= remaining {
			w.body.Write(b)
		} else {
			w.body.Write(b[:remaining])
			w.truncated = true
		}
	} else {
		w.truncated = true
	}
	return w.ResponseWriter.Write(b)
}

func (w *responseBodyWriter) WriteHeader(code int) {
	w.ResponseWriter.WriteHeader(code)
}

// capturedBody 返回捕获的响应体字符串
func (w *responseBodyWriter) capturedBody() string {
	return w.body.String()
}

// isTruncated 返回响应体是否被截断
func (w *responseBodyWriter) isTruncated() bool {
	return w.truncated
}

// sanitizeBody 脱敏敏感字段
func sanitizeBody(body string) string {
	if body == "" {
		return ""
	}

	// 简单的 JSON 字段脱敏：替换 password, token, secret, authorization 等字段的值
	sensitiveKeys := []string{
		`"password"`, `"token"`, `"secret"`, `"authorization"`,
		`"access_token"`, `"refresh_token"`, `"api_key"`, `"apikey"`,
	}

	result := body
	for _, key := range sensitiveKeys {
		result = sanitizeJSONField(result, key)
	}
	return result
}

// sanitizeJSONField 替换 JSON 中指定字段的值
func sanitizeJSONField(json, field string) string {
	idx := strings.Index(json, field)
	if idx < 0 {
		return json
	}

	// 找到字段后的冒号
	rest := json[idx+len(field):]
	colonIdx := strings.Index(rest, ":")
	if colonIdx < 0 {
		return json
	}

	// 找到值（跳过空白）
	valueStart := colonIdx + 1
	for valueStart < len(rest) && (rest[valueStart] == ' ' || rest[valueStart] == '\t') {
		valueStart++
	}

	if valueStart >= len(rest) {
		return json
	}

	var valueEnd int
	if rest[valueStart] == '"' {
		// 字符串值
		valueStart++
		valueEnd = strings.Index(rest[valueStart:], `"`)
		if valueEnd < 0 {
			return json
		}
		valueEnd += valueStart
		// 替换引号内的内容
		return json[:idx+len(field)+1+valueStart] + `***` + rest[valueEnd:]
	}

	// 非字符串值（数字、布尔、null）— 找到结束位置
	valueEnd = valueStart
	for valueEnd < len(rest) && rest[valueEnd] != ',' && rest[valueEnd] != '}' && rest[valueEnd] != '\n' {
		valueEnd++
	}

	return json[:idx+len(field)+1+valueStart] + `"***"` + rest[valueEnd:]
}
