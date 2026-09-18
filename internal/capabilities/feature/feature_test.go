package feature

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFlagIsEnabled(t *testing.T) {
	tests := []struct {
		name   string
		flag   Flag
		userID string
		want   bool
	}{
		{"disabled always false", Flag{Enabled: false, Percentage: 100}, "u1", false},
		{"whitelist explicit true", Flag{Enabled: true, Users: map[string]bool{"u1": true}}, "u1", true},
		{"whitelist explicit false", Flag{Enabled: true, Users: map[string]bool{"u1": false}}, "u1", false},
		{"whitelist missing falls back to percentage", Flag{Enabled: true, Users: map[string]bool{}, Percentage: 100}, "u9", true},
		{"percentage 100 always on", Flag{Enabled: true, Percentage: 100}, "u1", true},
		{"percentage 0 always off", Flag{Enabled: true, Percentage: 0}, "u1", false},
		{"percentage hash hit", Flag{Enabled: true, Percentage: 50}, "user-hash-hit", hashUserID("user-hash-hit")%100 < 50},
		{"empty userID no percentage", Flag{Enabled: true, Percentage: 50}, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.flag.IsEnabled(tt.userID))
		})
	}
}

func TestManagerLifecycle(t *testing.T) {
	m := NewManager()

	// 未注册返回 false
	assert.False(t, m.IsEnabled("missing"))
	_, ok := m.Get("missing")
	assert.False(t, ok)
	assert.False(t, m.Update("missing", func(f *Flag) { f.Enabled = true }))

	// 注册 + 覆盖重复注册
	m.Register(Flag{Name: "a", Enabled: true})
	m.Register(Flag{Name: "a", Enabled: false})
	assert.False(t, m.IsEnabled("a"))

	// 带 userID
	m.Register(Flag{Name: "b", Enabled: true, Percentage: 100})
	assert.True(t, m.IsEnabled("b", "u1"))

	// Get 返回副本值
	f, ok := m.Get("b")
	require.True(t, ok)
	assert.True(t, f.Enabled)
	assert.Equal(t, "b", f.Name)

	// List
	flags := m.List()
	assert.Len(t, flags, 2)

	// Update 生效
	assert.True(t, m.Update("a", func(f *Flag) { f.Enabled = true; f.Percentage = 100 }))
	assert.True(t, m.IsEnabled("a"))

	// context 注入/取出
	ctx := WithManager(context.Background(), m)
	got, ok := FromContext(ctx)
	require.True(t, ok)
	assert.Same(t, m, got)

	// 空 context 取不到
	_, ok = FromContext(context.Background())
	assert.False(t, ok)
}

func TestHashUserIDStable(t *testing.T) {
	assert.Equal(t, hashUserID("abc"), hashUserID("abc"))
	assert.NotEqual(t, hashUserID("a"), hashUserID("b"))
	// 负数取绝对值
	assert.GreaterOrEqual(t, hashUserID("zz"), 0)
}
