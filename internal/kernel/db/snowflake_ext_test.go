package db

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type stringKeyModel struct {
	ID   string `gorm:"primaryKey"`
	Name string
}

func TestRegisterSnowflakeHook_NilDB(t *testing.T) {
	RegisterSnowflakeHook(nil) // 不应 panic
}

func TestIsIntegerID(t *testing.T) {
	cases := []struct {
		name string
		typ  reflect.Type
		want bool
	}{
		{"uint64", reflect.TypeOf(uint64(0)), true},
		{"uint", reflect.TypeOf(uint(0)), true},
		{"uint32", reflect.TypeOf(uint32(0)), true},
		{"int64", reflect.TypeOf(int64(0)), true},
		{"int", reflect.TypeOf(int(0)), true},
		{"int32", reflect.TypeOf(int32(0)), true},
		{"string", reflect.TypeOf(""), false},
		{"bool", reflect.TypeOf(false), false},
		{"float64", reflect.TypeOf(float64(0)), false},
		{"int8", reflect.TypeOf(int8(0)), false}, // 不在支持列表
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, isIntegerID(&schema.Field{FieldType: c.typ}))
		})
	}
}

func TestSnowflakeHook_NonIntegerPK(t *testing.T) {
	require.NoError(t, InitSnowflake(1))
	defer func() { snowflakeGen = nil }()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	RegisterSnowflakeHook(db)
	require.NoError(t, db.AutoMigrate(&stringKeyModel{}))

	m := stringKeyModel{Name: "x"}
	require.NoError(t, db.Create(&m).Error)
	// 非整数主键跳过注入，ID 保持原值（sqlite 不自动填充）
	assert.Equal(t, "", m.ID)
}

func TestSnowflakeHook_PointerSlice(t *testing.T) {
	require.NoError(t, InitSnowflake(4))
	defer func() { snowflakeGen = nil }()

	db := openTestDB(t)
	ms := []*snowflakeModel{{Name: "a"}, {Name: "b"}}
	require.NoError(t, db.Create(&ms).Error)
	for _, m := range ms {
		assert.Greater(t, m.ID, uint64(1)<<40)
	}
	assert.NotEqual(t, ms[0].ID, ms[1].ID)
}

func TestSnowflakeHook_SinglePointerElement(t *testing.T) {
	require.NoError(t, InitSnowflake(5))
	defer func() { snowflakeGen = nil }()

	db := openTestDB(t)
	ptr := &snowflakeModel{Name: "ptr"}
	require.NoError(t, db.Create(ptr).Error)
	assert.Greater(t, ptr.ID, uint64(1)<<40)
}

func TestInitSnowflake_InvalidWorkerID(t *testing.T) {
	for _, wid := range []int64{-1, 1024, 4096} {
		t.Run(string(rune(wid)), func(t *testing.T) {
			require.Error(t, InitSnowflake(wid))
		})
	}
	require.NoError(t, InitSnowflake(1023))
	snowflakeGen = nil
}

func TestIndirect(t *testing.T) {
	x := 5
	p := &x
	pp := &p

	assert.Equal(t, x, indirect(reflect.ValueOf(pp)).Interface())
	assert.Equal(t, x, indirect(reflect.ValueOf(p)).Interface())
	assert.Equal(t, x, indirect(reflect.ValueOf(x)).Interface())

	// 接口
	var iface interface{} = x
	assert.Equal(t, x, indirect(reflect.ValueOf(&iface)).Interface())

	// nil 指针保持原样
	var np *int
	v := indirect(reflect.ValueOf(np))
	assert.Equal(t, reflect.Pointer, v.Kind())
	assert.True(t, v.IsNil())
}
