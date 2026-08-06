package middleware

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"jimu/internal/shared/errors"
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
)

// SignatureConfig API 签名验证配置
type SignatureConfig struct {
	Secret        []byte        // HMAC 密钥
	HeaderKey     string        // API Key Header，默认 "X-Api-Key"
	HeaderSign    string        // 签名 Header，默认 "X-Signature"
	HeaderTimestamp string      // 时间戳 Header，默认 "X-Timestamp"
	HeaderNonce   string        // 随机数 Header，默认 "X-Nonce"
	MaxAge        time.Duration // 签名有效期，默认 5 分钟
	Skipper       func(*gin.Context) bool
}

// DefaultSignatureConfig 返回默认签名配置
func DefaultSignatureConfig(secret []byte) SignatureConfig {
	return SignatureConfig{
		Secret:          secret,
		HeaderKey:       "X-Api-Key",
		HeaderSign:      "X-Signature",
		HeaderTimestamp: "X-Timestamp",
		HeaderNonce:     "X-Nonce",
		MaxAge:          5 * time.Minute,
	}
}

// Signature API 请求签名验证中间件
// 签名算法：HMAC-SHA256(secret, method + path + query_sorted + body + timestamp + nonce)
func Signature(cfg SignatureConfig) gin.HandlerFunc {
	if cfg.MaxAge == 0 {
		cfg.MaxAge = 5 * time.Minute
	}

	return func(c *gin.Context) {
		if cfg.Skipper != nil && cfg.Skipper(c) {
			c.Next()
			return
		}

		// 读取必要 Header
		apiKey := c.GetHeader(cfg.HeaderKey)
		signature := c.GetHeader(cfg.HeaderSign)
		timestampStr := c.GetHeader(cfg.HeaderTimestamp)
		nonce := c.GetHeader(cfg.HeaderNonce)

		if apiKey == "" || signature == "" || timestampStr == "" || nonce == "" {
			response.Fail(c, errors.New(errors.CodeUnauthorized, "missing signature headers"))
			c.Abort()
			return
		}

		// 校验时间戳（防重放）
		timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
		if err != nil {
			response.Fail(c, errors.New(errors.CodeUnauthorized, "invalid timestamp"))
			c.Abort()
			return
		}

		ts := time.Unix(timestamp, 0)
		if time.Since(ts) > cfg.MaxAge || time.Since(ts) < -cfg.MaxAge {
			response.Fail(c, errors.New(errors.CodeUnauthorized, "signature expired"))
			c.Abort()
			return
		}

		// 构建签名字符串
		var body []byte
		if c.Request.Body != nil {
			body, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
		}

		signStr := buildSignString(
			c.Request.Method,
			c.Request.URL.Path,
			c.Request.URL.Query(),
			body,
			timestampStr,
			nonce,
		)

		// 计算期望签名
		expected := hmacSign(cfg.Secret, signStr)

		if !hmac.Equal([]byte(signature), []byte(expected)) {
			response.Fail(c, errors.New(errors.CodeUnauthorized, "invalid signature"))
			c.Abort()
			return
		}

		c.Next()
	}
}

// buildSignString 构建待签名字符串
func buildSignString(method, path string, query map[string][]string, body []byte, timestamp, nonce string) string {
	var sb strings.Builder

	// 1. METHOD + PATH
	sb.WriteString(strings.ToUpper(method))
	sb.WriteString(path)

	// 2. 排序后的 Query 参数
	if len(query) > 0 {
		keys := make([]string, 0, len(query))
		for k := range query {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			vals := query[k]
			sort.Strings(vals)
			for _, v := range vals {
				sb.WriteString(k)
				sb.WriteString(v)
			}
		}
	}

	// 3. Body
	sb.Write(body)

	// 4. Timestamp + Nonce
	sb.WriteString(timestamp)
	sb.WriteString(nonce)

	return sb.String()
}

// hmacSign 计算 HMAC-SHA256 签名
func hmacSign(secret []byte, data string) string {
	h := hmac.New(sha256.New, secret)
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// SignRequest 用于客户端生成签名（测试/SDK 用）
func SignRequest(secret []byte, method, path string, query map[string][]string, body []byte, timestamp int64, nonce string) string {
	signStr := buildSignString(method, path, query, body, strconv.FormatInt(timestamp, 10), nonce)
	return hmacSign(secret, signStr)
}

// 确保导入完整
var _ = http.MethodGet
