// Package memory 是 cache 能力的进程内驱动。
//
// 单实例语义：每个进程各自持有一份缓存，多实例部署时实例间不共享。
// 适合 0 依赖项目和测试；需要跨实例一致性的场景应选 redis 驱动。
package memory

import (
	"jimu/internal/capabilities/cache"
	"jimu/internal/platform/cachemem"
)

func init() { cache.Register("memory", newFromOptions) }

func newFromOptions(opts cache.Options) (cache.Cache, error) {
	return cachemem.New(opts.Prefix), nil
}
