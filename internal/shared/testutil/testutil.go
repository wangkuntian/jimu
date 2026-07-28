package testutil

import (
	"fmt"
	"math/rand"
	"time"
)

// RandomString 生成随机字符串
func RandomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// RandomEmail 生成随机邮箱
func RandomEmail() string {
	return fmt.Sprintf("%s@example.com", RandomString(10))
}

// RandomMobile 生成随机手机号
func RandomMobile() string {
	prefixes := []string{"138", "139", "150", "151", "158", "159", "188", "189"}
	prefix := prefixes[rand.Intn(len(prefixes))]
	suffix := fmt.Sprintf("%08d", rand.Intn(100000000))
	return prefix + suffix
}

// RandomInt 生成指定范围的随机整数
func RandomInt(min, max int) int {
	return min + rand.Intn(max-min+1)
}

// Ptr 返回指针
func Ptr[T any](v T) *T {
	return &v
}

// Now 返回当前时间（测试用）
func Now() time.Time {
	return time.Now().UTC()
}
