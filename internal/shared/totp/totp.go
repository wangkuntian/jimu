// Package totp 实现 RFC 6238 TOTP（基于时间的一次性密码）。
// 算法：HMAC-SHA1(secret, time_counter)，取动态截断 31 位整数模 10^digits，
// 默认 6 位、30 秒时间步长、允许前/后各 1 个时间窗口的时钟偏移。
package totp

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"strings"
	"time"
)

const (
	// DefaultPeriod 默认时间步长（秒）
	DefaultPeriod = 30
	// DefaultDigits 默认验证码位数
	DefaultDigits = 6
	// DefaultSkew 默认允许的时钟偏移窗口数（前/后各 N 个窗口）
	DefaultSkew = 1
)

// Secret 生成新的随机 TOTP 密钥（20 字节 SHA-1 长度，base32 无填充编码）。
func Secret() (string, error) {
	buf := make([]byte, 20)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("totp: generate secret: %w", err)
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(buf), nil
}

// Code 计算指定时间对应的验证码。
func Code(secret string, t time.Time, period, digits int) (string, error) {
	key, err := decodeSecret(secret)
	if err != nil {
		return "", err
	}
	if period <= 0 {
		period = DefaultPeriod
	}
	if digits <= 0 {
		digits = DefaultDigits
	}
	counter := uint64(t.Unix() / int64(period))
	return generate(key, counter, digits), nil
}

// Validate 校验验证码，允许前后各 skew 个时间窗口的时钟偏移。
// 空 secret 或空 code 视为无效。
func Validate(secret, code string, t time.Time, period, digits, skew int) bool {
	if secret == "" || code == "" {
		return false
	}
	key, err := decodeSecret(secret)
	if err != nil {
		return false
	}
	if period <= 0 {
		period = DefaultPeriod
	}
	if digits <= 0 {
		digits = DefaultDigits
	}
	if skew < 0 {
		skew = DefaultSkew
	}
	counter := t.Unix() / int64(period)
	for i := -skew; i <= skew; i++ {
		if hmacEqual(code, generate(key, uint64(counter+int64(i)), digits)) {
			return true
		}
	}
	return false
}

// ProvisioningURI 生成 otpauth:// URI，供二维码/认证器 App 扫描绑定。
// issuer 为服务名称（如 jimu），account 通常为用户名或邮箱。
func ProvisioningURI(secret, account, issuer string) string {
	u := fmt.Sprintf("otpauth://totp/%s?secret=%s&issuer=%s&algorithm=SHA1&digits=%d&period=%d",
		urlEscape(account), secret, urlEscape(issuer), DefaultDigits, DefaultPeriod)
	return u
}

func decodeSecret(secret string) ([]byte, error) {
	s := strings.ToUpper(strings.TrimSpace(secret))
	s = strings.TrimRight(s, "=")
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("totp: invalid secret: %w", err)
	}
	return key, nil
}

// generate 计算 HMAC-SHA1 动态口令（RFC 4226 HOTP + RFC 6238 TOTP）。
func generate(key []byte, counter uint64, digits int) string {
	var msg [8]byte
	binary.BigEndian.PutUint64(msg[:], counter)

	mac := hmac.New(sha1.New, key)
	mac.Write(msg[:])
	hash := mac.Sum(nil)

	// 动态截断：取最后一个字节的低 4 位作为偏移
	offset := hash[len(hash)-1] & 0x0f
	code := (uint32(hash[offset])&0x7f)<<24 |
		(uint32(hash[offset+1])&0xff)<<16 |
		(uint32(hash[offset+2])&0xff)<<8 |
		uint32(hash[offset+3])&0xff

	mod := uint32(1)
	for i := 0; i < digits; i++ {
		mod *= 10
	}
	return fmt.Sprintf("%0*d", digits, code%mod)
}

// hmacEqual 常数时间比较（避免时序侧信道）。
func hmacEqual(a, b string) bool {
	return hmac.Equal([]byte(a), []byte(b))
}

func urlEscape(s string) string {
	replacer := strings.NewReplacer(" ", "%20", ":", "%3A", "+", "%2B", "/", "%2F", "?", "%3F")
	return replacer.Replace(s)
}
