package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResponseBodyWriterCapturesAndTruncates(t *testing.T) {
	w := httptest.NewRecorder()
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(w)

	bw := newResponseBodyWriter(c.Writer, 8)
	require.NotNil(t, bw)

	_, err := bw.Write([]byte("hello"))
	require.NoError(t, err)
	assert.Equal(t, "hello", bw.capturedBody())
	assert.False(t, bw.isTruncated())

	// 超过 maxSize 的部分被截断但标记
	_, err = bw.Write([]byte("world-extra"))
	require.NoError(t, err)
	assert.Equal(t, "hellowor", bw.capturedBody())
	assert.True(t, bw.isTruncated())

	// 已满后再写仍标记截断
	_, err = bw.Write([]byte("x"))
	require.NoError(t, err)
	assert.True(t, bw.isTruncated())

	// 底层响应体完整透传
	assert.Equal(t, "helloworld-extrax", w.Body.String())
}

func TestResponseBodyWriterWriteHeaderPassthrough(t *testing.T) {
	w := httptest.NewRecorder()
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(w)

	bw := newResponseBodyWriter(c.Writer, 1024)
	bw.WriteHeader(http.StatusCreated)
	// gin 的 responseWriter 延迟写头：状态记录在 writer，写 body 后落到 recorder
	assert.Equal(t, http.StatusCreated, c.Writer.Status())
	_, err := bw.Write([]byte("x"))
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "x", w.Body.String())
}

func TestSanitizeBody(t *testing.T) {
	// 注意：当前实现保留敏感值第一个字符，仅掩蔽剩余部分
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"string sensitive fields", `{"password":"p","token":"t","secret":"s"}`, `{"password":"p***","token":"t***","secret":"s***"}`},
		{"numeric secret", `{"secret":123}`, `{"secret":1"***"}`},
		{"non-sensitive untouched", `{"count":123,"name":"x"}`, `{"count":123,"name":"x"}`},
		{"nested access token", `{"access_token":"abc","api_key":"k"}`, `{"access_token":"a***","api_key":"k***"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, sanitizeBody(tt.in))
		})
	}
}

func TestSanitizeJSONFieldEdgeCases(t *testing.T) {
	// 字段不存在
	assert.Equal(t, `{"a":1}`, sanitizeJSONField(`{"a":1}`, `"password"`))
	// 字段后无冒号
	assert.Equal(t, `"password"`, sanitizeJSONField(`"password"`, `"password"`))
	// 字符串值未闭合
	assert.Equal(t, `{"password":"abc`, sanitizeJSONField(`{"password":"abc`, `"password"`))
	// 字段后只有冒号没有值
	assert.Equal(t, `{"password":`, sanitizeJSONField(`{"password":`, `"password"`))
}
