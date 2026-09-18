package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func sampleLog() AuditLog {
	return AuditLog{
		TenantID:  1,
		UserID:    2,
		Username:  "alice",
		Action:    "create",
		Resource:  "/api/v1/users",
		Detail:    "created user",
		IP:        "1.2.3.4",
		Method:    "POST",
		Path:      "/api/v1/users",
		Status:    201,
		CreatedAt: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
	}
}

func TestComputeEntryHashIsDeterministic(t *testing.T) {
	l := sampleLog()
	h1 := l.ComputeEntryHash(nil)
	h2 := l.ComputeEntryHash(nil)

	assert.Len(t, h1, 64, "应为 sha256 十六进制摘要")
	assert.Equal(t, h1, h2)
}

func TestComputeEntryHashDetectsContentChange(t *testing.T) {
	l := sampleLog()
	h := l.ComputeEntryHash(nil)

	tampered := l
	tampered.Action = "delete"
	assert.NotEqual(t, h, tampered.ComputeEntryHash(nil), "内容变化必须改变哈希")

	tampered = l
	tampered.ChangesRaw = `[{"field":"status"}]`
	assert.NotEqual(t, h, tampered.ComputeEntryHash(nil), "变更明细变化必须改变哈希")
}

func TestComputeEntryHashDependsOnPrevHash(t *testing.T) {
	l := sampleLog()
	h := l.ComputeEntryHash(nil)

	linked := l
	linked.PrevHash = h
	assert.NotEqual(t, h, linked.ComputeEntryHash(nil), "链接位置变化必须改变哈希")
}

func TestComputeEntryHashUsesHMACWhenSecretSet(t *testing.T) {
	l := sampleLog()
	assert.NotEqual(t, l.ComputeEntryHash(nil), l.ComputeEntryHash([]byte("secret")))
	assert.Equal(t, l.ComputeEntryHash([]byte("secret")), l.ComputeEntryHash([]byte("secret")))
}

func TestComputeEntryHashTruncatesToSecond(t *testing.T) {
	l := sampleLog()
	withMillis := l
	withMillis.CreatedAt = l.CreatedAt.Add(999 * time.Millisecond)

	// 数据库 TIMESTAMP 精度为秒，毫秒差异不应影响哈希
	assert.Equal(t, l.ComputeEntryHash(nil), withMillis.ComputeEntryHash(nil))
}

func TestMatchesEntryHash(t *testing.T) {
	l := sampleLog()
	l.EntryHash = l.ComputeEntryHash(nil)
	assert.True(t, l.MatchesEntryHash(nil))

	l.Action = "delete"
	assert.False(t, l.MatchesEntryHash(nil), "内容被改动后校验应失败")

	l = sampleLog()
	assert.False(t, l.MatchesEntryHash(nil), "未哈希条目不应通过校验")
}

func TestNormalizeCreatedAt(t *testing.T) {
	cst := time.FixedZone("CST", 8*3600)
	got := NormalizeCreatedAt(time.Date(2026, 1, 2, 11, 4, 5, 123456789, cst))

	assert.Equal(t, time.UTC, got.Location())
	assert.Zero(t, got.Nanosecond())
	assert.Equal(t, time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC), got)
}
