// internal/platform/db/encryption.go
package db

import (
	"reflect"
	"strings"

	"jimu/internal/platform/encryption"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// RegisterEncryptionHooks 注册字段级加密回调：
//   - 写入（Create/Update）：对带 `encryption:"true"` tag 的字段加密落库；
//     对带 `blind:"<field>"` tag 的字段用对应明文计算确定性盲索引（唯一约束/精确查询用）。
//   - 读取（Query/Update 后）：对 `encryption:"true"` 字段解密回内存明文。
//
// cipher 为明文模式（密钥空）时透传，功能等价于不加密，但盲索引仍计算，
// 保证 email_hash/phone_hash 唯一约束与查找始终可用。
func RegisterEncryptionHooks(g *gorm.DB, c *encryption.Cipher) {
	if g == nil || c == nil {
		return
	}
	blind := func(d *gorm.DB) {
		applyBlindIndexFields(d, func(f *schema.Field, sourceValue string, v reflect.Value) {
			if strings.TrimSpace(sourceValue) == "" {
				// 源字段为空：hash 置零值（*string→NULL，string→""）。
				// NULL 不受唯一索引约束，可选 email/phone 的多用户可共存。
				v.Set(reflect.Zero(v.Type()))
				return
			}
			hash := c.BlindIndex(sourceValue)
			if v.Kind() == reflect.Pointer {
				pv := reflect.New(v.Type().Elem())
				pv.Elem().SetString(hash)
				v.Set(pv)
				return
			}
			v.SetString(hash)
		})
	}
	encrypt := func(d *gorm.DB) {
		applyEncryptedFields(d, func(v reflect.Value) {
			enc, err := c.Encrypt(v.String())
			if err != nil {
				_ = d.AddError(err)
				return
			}
			v.SetString(enc)
		})
	}
	// 先盲索引（读明文）再加密（覆盖密文），两回调按注册顺序执行
	_ = g.Callback().Create().Before("gorm:create").Register("encryption:blind_index", blind)
	_ = g.Callback().Create().Before("gorm:create").Register("encryption:encrypt", encrypt)
	_ = g.Callback().Update().Before("gorm:update").Register("encryption:blind_index", blind)
	_ = g.Callback().Update().Before("gorm:update").Register("encryption:encrypt", encrypt)

	decrypt := func(d *gorm.DB) {
		applyEncryptedFields(d, func(v reflect.Value) {
			dec, err := c.Decrypt(v.String())
			if err != nil {
				_ = d.AddError(err)
				return
			}
			v.SetString(dec)
		})
	}
	_ = g.Callback().Query().After("gorm:query").Register("encryption:decrypt", decrypt)
	_ = g.Callback().Update().After("gorm:update").Register("encryption:decrypt", decrypt)
}

// applyEncryptedFields 遍历写入/读出的每个元素，对带 encryption:"true" tag 的字段执行 fn。
func applyEncryptedFields(d *gorm.DB, fn func(v reflect.Value)) {
	if d.Statement.Schema == nil {
		return
	}
	each := func(elem reflect.Value) {
		for _, f := range d.Statement.Schema.Fields {
			if tag, ok := f.Tag.Lookup("encryption"); ok && tag == "true" {
				if fv := f.ReflectValueOf(d.Statement.Context, elem); fv.CanSet() {
					fn(fv)
				}
			}
		}
	}
	walkElements(d, each)
}

// applyBlindIndexFields 遍历写入的每个元素，对带 blind:"<source>" tag 的字段，
// 用 source 字段的当前明文值计算盲索引写入（保持与加密字段一致）。
func applyBlindIndexFields(d *gorm.DB, fn func(f *schema.Field, sourceValue string, v reflect.Value)) {
	if d.Statement.Schema == nil {
		return
	}
	each := func(elem reflect.Value) {
		for _, f := range d.Statement.Schema.Fields {
			sourceName, ok := f.Tag.Lookup("blind")
			if !ok || sourceName == "" {
				continue
			}
			src := d.Statement.Schema.LookUpField(sourceName)
			if src == nil {
				continue
			}
			sv := src.ReflectValueOf(d.Statement.Context, elem)
			if fv := f.ReflectValueOf(d.Statement.Context, elem); fv.CanSet() {
				fn(f, sv.String(), fv)
			}
		}
	}
	walkElements(d, each)
}

// walkElements 遍历 d.Statement.ReflectValue（结构体/指针或批量 slice），逐元素调用 fn。
// 用 ReflectValueOf 而非 Field.Set：后者对批量 slice 元素走 fallbackSetter 的类型转换路径会失败。
func walkElements(d *gorm.DB, fn func(v reflect.Value)) {
	rv := d.Statement.ReflectValue
	switch rv.Kind() {
	case reflect.Struct, reflect.Pointer:
		fn(indirect(rv))
	case reflect.Slice, reflect.Array:
		for i := 0; i < rv.Len(); i++ {
			fn(indirect(rv.Index(i)))
		}
	}
}
