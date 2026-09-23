package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRejectsUnregisteredDriver(t *testing.T) {
	assert.Empty(t, RegisteredTypes(), "核心包不得自注册任何驱动")
	_, err := New(Config{Type: StorageTypeS3})
	require.ErrorContains(t, err, `storage driver "s3" is not compiled into this build`)
	require.ErrorContains(t, err, "compiled: none")
}

func TestNewUsesRegisteredDriverAndEmptyTypeFallsBackToLocal(t *testing.T) {
	Register("fake", func(cfg Config) (Storage, error) { return nil, nil })
	// 「空类型按 local 处理」的断言必须锁定 local 这个具体取值：只断言错误文案含
	// "not compiled into this build" 没有区分力 —— 删掉 factory.go 的空类型 fallback 后，
	// 文案仅由 `storage driver "local" ...` 变成 `storage driver "" ...`，宽泛断言照样通过。
	// 因此这里给 StorageTypeLocal 注册一个 sentinel 工厂，验证空类型确实被归一化为 local
	// 并命中该工厂（工厂内断言收到的 cfg.Type），缺了 fallback 时 New 先报未注册错误即红。
	localCalled := false
	Register(StorageTypeLocal, func(cfg Config) (Storage, error) {
		localCalled = true
		require.Equal(t, StorageTypeLocal, cfg.Type, "空类型必须先归一化为 local 再交给驱动工厂")
		return nil, nil
	})
	t.Cleanup(func() {
		delete(drivers, "fake")
		delete(drivers, StorageTypeLocal)
	})

	got, err := New(Config{Type: "fake"})
	require.NoError(t, err)
	var want Storage
	assert.Equal(t, want, got)

	got, err = New(Config{}) // 空类型按 local 处理
	require.NoError(t, err, "空类型必须命中 local 驱动，而不是报未编译进本构建")
	assert.Nil(t, got)
	assert.True(t, localCalled, "空类型必须路由到 local 驱动工厂")
}

func TestRegisterPanicsOnDuplicate(t *testing.T) {
	Register("dup", func(Config) (Storage, error) { return nil, nil })
	t.Cleanup(func() { delete(drivers, "dup") })
	assert.Panics(t, func() { Register("dup", func(Config) (Storage, error) { return nil, nil }) })
}
