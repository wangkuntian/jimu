// internal/platform/db/snowflake.go
package db

import (
	"reflect"

	"jimu/internal/shared/id"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// snowflakeGen 全局雪花生成器。InitSnowflake 初始化后启用主键注入。
var snowflakeGen id.Generator

// InitSnowflake 设置全局雪花生成器，必须在 ConnectWithRetry 之前调用。
// workerID 范围 0-1023，多实例部署时每个副本需唯一，避免同毫秒 ID 冲突。
func InitSnowflake(workerID int64) error {
	gen, err := id.NewSnowflake(workerID)
	if err != nil {
		return err
	}
	snowflakeGen = gen
	return nil
}

// RegisterSnowflakeHook 注册 BeforeCreate 回调：主键为零值时注入雪花 ID。
// snowflakeGen 未初始化时跳过（回退数据库自增），保证测试等入口不依赖 InitSnowflake。
func RegisterSnowflakeHook(g *gorm.DB) {
	if g == nil {
		return
	}
	g.Callback().Create().Before("gorm:create").Register("snowflake:assign_id", func(d *gorm.DB) {
		gen := snowflakeGen
		if gen == nil || d.Statement.Schema == nil {
			return
		}
		f := d.Statement.Schema.PrioritizedPrimaryField
		if f == nil || !isIntegerID(f) {
			return
		}

		// 结构体/指针或批量 Slice，统一按元素注入。
		// 用 ReflectValueOf + Set 而非 Field.Set：后者对批量 slice 元素走
		// fallbackSetter 的类型转换路径会失败（gorm 字段类型推断问题）。
		assign := func(v reflect.Value) {
			fv := f.ReflectValueOf(d.Statement.Context, indirect(v))
			if !fv.IsZero() {
				return
			}
			nid, err := gen.NextID()
			if err != nil {
				d.AddError(err)
				return
			}
			fv.Set(reflect.ValueOf(nid).Convert(fv.Type()))
		}

		rv := d.Statement.ReflectValue
		switch rv.Kind() {
		case reflect.Struct, reflect.Ptr:
			assign(rv)
		case reflect.Slice, reflect.Array:
			for i := 0; i < rv.Len(); i++ {
				assign(rv.Index(i))
			}
		}
	})
}

// isIntegerID 主键是否为整数类型（雪花 ID 为 uint64，兼容 int64/uint）
func isIntegerID(f *schema.Field) bool {
	switch f.FieldType.Kind() {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Int, reflect.Int32, reflect.Int64:
		return true
	}
	return false
}

// indirect 解引用指针/接口，返回底层值（与 gorm 内部一致）
func indirect(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return v
		}
		v = v.Elem()
	}
	return v
}
