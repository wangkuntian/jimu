package encryption

import (
	"fmt"

	"jimu/internal/assembly"
	"jimu/internal/contract"
)

// Wire 装配字段级加密能力：构造 Cipher 并注册全局 gorm hook（加密 email/phone 写入 +
// 盲索引 + 读取解密），把 *Cipher 暴露为端口供 user/auth 消费（端口缺失即不加密降级）。
func Wire(ctx *assembly.Context) (contract.Module, error) {
	cipher := New(ctx.Config().Security.EncryptionKey)
	RegisterHooks(ctx.DB(), cipher)
	if err := ctx.Provide(PortName, cipher); err != nil {
		return nil, fmt.Errorf("provide encryption port: %w", err)
	}
	return nil, nil
}
