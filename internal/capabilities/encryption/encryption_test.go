package encryption

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testKey = "0123456789abcdef0123456789abcdef" // 32 字节

func TestEncryptDecryptRoundtrip(t *testing.T) {
	c := New(testKey)
	enc, err := c.Encrypt("user@example.com")
	require.NoError(t, err)
	assert.NotEqual(t, "user@example.com", enc)
	assert.True(t, strings.HasPrefix(enc, "enc:v1:"))

	dec, err := c.Decrypt(enc)
	require.NoError(t, err)
	assert.Equal(t, "user@example.com", dec)
}

func TestEncryptProducesRandomCiphertext(t *testing.T) {
	c := New(testKey)
	a, _ := c.Encrypt("same@example.com")
	b, _ := c.Encrypt("same@example.com")
	assert.NotEqual(t, a, b, "随机 nonce：同一明文两次加密密文必须不同")
}

func TestDecryptTamperedFails(t *testing.T) {
	c := New(testKey)
	enc, err := c.Encrypt("user@example.com")
	require.NoError(t, err)
	tampered := enc[:len(enc)-4] + "zzzz"
	_, err = c.Decrypt(tampered)
	assert.Error(t, err)
}

func TestDecryptWrongKeyFails(t *testing.T) {
	enc, err := New(testKey).Encrypt("secret")
	require.NoError(t, err)
	_, err = New("fedcba9876543210fedcba9876543210").Decrypt(enc)
	assert.Error(t, err)
}

func TestPlaintextPassthrough(t *testing.T) {
	c := New(testKey)
	// 历史明文/明文模式存量行：无 enc:v1: 前缀原样返回
	dec, err := c.Decrypt("legacy@example.com")
	require.NoError(t, err)
	assert.Equal(t, "legacy@example.com", dec)
}

func TestEmptyValuePassthrough(t *testing.T) {
	c := New(testKey)
	enc, err := c.Encrypt("")
	require.NoError(t, err)
	assert.Equal(t, "", enc)
	dec, err := c.Decrypt("")
	require.NoError(t, err)
	assert.Equal(t, "", dec)
}

func TestNoKeyPlaintextMode(t *testing.T) {
	c := New("")
	enc, err := c.Encrypt("user@example.com")
	require.NoError(t, err)
	assert.Equal(t, "user@example.com", enc, "无密钥时明文透传")
	dec, err := c.Decrypt(enc)
	require.NoError(t, err)
	assert.Equal(t, "user@example.com", dec)
}

func TestNoKeyCannotDecryptCiphertext(t *testing.T) {
	enc, err := New(testKey).Encrypt("secret")
	require.NoError(t, err)
	_, err = New("").Decrypt(enc)
	assert.Error(t, err, "无密钥时遇到已加密数据必须报错而非静默返回")
}

func TestBlindIndexDeterministicAndDistinct(t *testing.T) {
	c := New(testKey)
	a := c.BlindIndex("user@example.com")
	b := c.BlindIndex("user@example.com")
	c2 := c.BlindIndex("other@example.com")
	assert.Equal(t, a, b, "盲索引必须确定性")
	assert.NotEqual(t, a, c2)
	assert.Equal(t, 64, len(a))
}

func TestBlindIndexWithoutKey(t *testing.T) {
	a := New("").BlindIndex("user@example.com")
	b := New("").BlindIndex("user@example.com")
	assert.Equal(t, a, b, "无钥盲索引（SHA-256）仍确定性")
}

func TestBlindIndexEmpty(t *testing.T) {
	assert.Equal(t, "", New(testKey).BlindIndex(""))
}
