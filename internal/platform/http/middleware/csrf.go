package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"jimu/internal/shared/errors"
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
)

// CSRFConfig CSRF 防护配置
type CSRFConfig struct {
	Secret       []byte                  // HMAC 密钥
	TokenHeader  string                  // Token 所在 Header，默认 "X-CSRF-Token"
	TokenField   string                  // Token 所在 Form Field，默认 "_csrf"
	CookieName   string                  // Cookie 名称，默认 "csrf_token"
	CookieMaxAge int                     // Cookie 有效期（秒），默认 86400
	SafeMethods  []string                // 安全方法（不校验），默认 GET/HEAD/OPTIONS
	Skipper      func(*gin.Context) bool // 跳过校验的路径判断
}

// DefaultCSRFConfig 返回默认 CSRF 配置
func DefaultCSRFConfig(secret []byte) CSRFConfig {
	return CSRFConfig{
		Secret:       secret,
		TokenHeader:  "X-CSR-Token",
		TokenField:   "_csrf",
		CookieName:   "csrf_token",
		CookieMaxAge: 86400,
		SafeMethods:  []string{"GET", "HEAD", "OPTIONS"},
	}
}

// generateToken 生成 CSRF Token（HMAC(secret, timestamp)）
func generateToken(secret []byte, timestamp int64) string {
	h := hmac.New(sha256.New, secret)
	h.Write([]byte(string(rune(timestamp))))
	return hex.EncodeToString(h.Sum(nil))
}

// validateToken 校验 CSRF Token
func validateToken(secret []byte, token string, timestamp int64, maxAge time.Duration) bool {
	if token == "" {
		return false
	}
	// 检查时间戳是否过期
	if time.Since(time.Unix(timestamp, 0)) > maxAge {
		return false
	}
	expected := generateToken(secret, timestamp)
	return hmac.Equal([]byte(token), []byte(expected))
}

// CSRF CSRF 防护中间件
// 使用 double-submit cookie pattern：token 同时存在于 cookie 和 header/form 中
func CSRF(cfg CSRFConfig) gin.HandlerFunc {
	if len(cfg.SafeMethods) == 0 {
		cfg.SafeMethods = []string{"GET", "HEAD", "OPTIONS"}
	}

	return func(c *gin.Context) {
		// 安全方法只设置 token，不校验
		if isSafeMethod(c.Request.Method, cfg.SafeMethods) {
			setCSRFCookie(c, cfg)
			c.Next()
			return
		}

		// 跳过指定路径
		if cfg.Skipper != nil && cfg.Skipper(c) {
			c.Next()
			return
		}

		// 使用 Bearer token 认证的请求跳过 CSRF（API 场景）
		authHeader := c.GetHeader("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			c.Next()
			return
		}

		// 从 cookie 中读取 token
		cookieToken, err := c.Cookie(cfg.CookieName)
		if err != nil || cookieToken == "" {
			response.Fail(c, errors.New(errors.CodeForbidden, "CSRF token missing"))
			c.Abort()
			return
		}

		// 从 header 或 form 中读取 token
		var requestToken string
		if cfg.TokenHeader != "" {
			requestToken = c.GetHeader(cfg.TokenHeader)
		}
		if requestToken == "" && cfg.TokenField != "" {
			requestToken = c.PostForm(cfg.TokenField)
		}

		if requestToken == "" {
			response.Fail(c, errors.New(errors.CodeForbidden, "CSRF token missing"))
			c.Abort()
			return
		}

		// 比对两个 token
		if !hmac.Equal([]byte(cookieToken), []byte(requestToken)) {
			response.Fail(c, errors.New(errors.CodeForbidden, "CSRF token invalid"))
			c.Abort()
			return
		}

		c.Next()
	}
}

// CSRFSigner 用于生成和校验 CSRF Token 的辅助工具
type CSRFSigner struct {
	secret []byte
}

// NewCSRFSigner 创建 CSRF Signer
func NewCSRFSigner(secret []byte) *CSRFSigner {
	return &CSRFSigner{secret: secret}
}

// Generate 生成 token + timestamp
func (s *CSRFSigner) Generate() (token string, timestamp int64) {
	timestamp = time.Now().Unix()
	token = generateToken(s.secret, timestamp)
	return
}

// Validate 校验 token
func (s *CSRFSigner) Validate(token string, timestamp int64, maxAge time.Duration) bool {
	return validateToken(s.secret, token, timestamp, maxAge)
}

// setCSRFCookie 设置 CSRF Cookie
func setCSRFCookie(c *gin.Context, cfg CSRFConfig) {
	// 如果已有有效 cookie 则不重新设置
	if _, err := c.Cookie(cfg.CookieName); err == nil {
		return
	}

	timestamp := time.Now().Unix()
	token := generateToken(cfg.Secret, timestamp)

	c.SetCookie(
		cfg.CookieName,
		token,
		cfg.CookieMaxAge,
		"/",
		"",
		false, // Secure: 生产环境应为 true
		true,  // HttpOnly
	)
}

func isSafeMethod(method string, safeMethods []string) bool {
	for _, m := range safeMethods {
		if strings.EqualFold(method, m) {
			return true
		}
	}
	return false
}
