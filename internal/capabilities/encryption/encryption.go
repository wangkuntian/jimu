// Package encryption 提供字段级敏感数据加密：AES-256-GCM + 确定性盲索引。
//
// 密钥为空时退化为明文模式（Encrypt/BlindIndex 直接透传/无钥哈希），
// 保证未配置加密密钥的环境功能始终可用；配置后存量明文行仍可解密读回。
package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

// prefix 密文版本前缀。解密时无此前缀的值视为明文原样返回，
// 兼容历史数据与明文模式；轮换密钥时升级前缀即可（ponytail: 轮换机制见 README）。
const prefix = "enc:v1:"

// Cipher AES-256-GCM 字段加密器。key 为空表示明文模式。
type Cipher struct {
	key []byte
}

// New 创建加密器。key 为空时返回明文模式的 Cipher（Encrypt 透传）。
// 生产环境校验见 config.validate（encryption_key 非空需 ≥32 字节）。
func New(key string) *Cipher {
	return &Cipher{key: []byte(key)}
}

// Encrypt 加密字符串。key 为空时原样返回（明文模式）。
func (c *Cipher) Encrypt(s string) (string, error) {
	if s == "" || len(c.key) == 0 {
		return s, nil
	}
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", fmt.Errorf("aes new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("aes gcm: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("nonce: %w", err)
	}
	sealed := gcm.Seal(nonce, nonce, []byte(s), nil)
	return prefix + base64.StdEncoding.EncodeToString(sealed), nil
}

// Decrypt 解密字符串。值非 enc:v1: 前缀时视为明文原样返回（历史数据/明文模式兼容）。
func (c *Cipher) Decrypt(s string) (string, error) {
	if s == "" || !strings.HasPrefix(s, prefix) {
		return s, nil
	}
	if len(c.key) == 0 {
		return "", errors.New("cannot decrypt encrypted value without encryption key")
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(s, prefix))
	if err != nil {
		return "", fmt.Errorf("decode ciphertext: %w", err)
	}
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", fmt.Errorf("aes new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("aes gcm: %w", err)
	}
	if len(raw) < gcm.NonceSize() {
		return "", errors.New("ciphertext too short")
	}
	nonce, sealed := raw[:gcm.NonceSize()], raw[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, sealed, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt: %w", err)
	}
	return string(plain), nil
}

// BlindIndex 计算确定性盲索引，用于唯一约束与精确查询（如 FindByEmailHash）。
// 有密钥时 HMAC-SHA256（标准 blind index），无密钥时退化为 SHA-256。
func (c *Cipher) BlindIndex(s string) string {
	if s == "" {
		return ""
	}
	if len(c.key) > 0 {
		mac := hmac.New(sha256.New, c.key)
		mac.Write([]byte(s))
		return hex.EncodeToString(mac.Sum(nil))
	}
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
