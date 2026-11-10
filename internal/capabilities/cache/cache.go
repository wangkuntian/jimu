// Package cache 定义缓存能力：缓存语义（what）与具体实现（how）分离。
// 实现由驱动包提供（redis/memory），驱动在 init() 中向本包注册。
package cache

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"jimu/internal/contract"
)

// Cache 是缓存能力的完整接口：在能力间共享的 contract.Cache 之上扩展缓存专属操作。
// 跨能力消费方只依赖 contract.Cache，能力内部和驱动才用完整接口。
type Cache interface {
	contract.Cache
	DeletePattern(ctx context.Context, pattern string) error
	Exists(ctx context.Context, key string) (bool, error)
	GetOrSet(ctx context.Context, key string, dest interface{}, ttl time.Duration, fetch func() (interface{}, error)) error
	Close() error
}

// Factory 是驱动的构造签名字面量。具体参数由各驱动自行定义并在 init() 中注册闭包。
type Factory func(Options) (Cache, error)

// Options 是构造缓存实例的通用参数。
type Options struct {
	// Prefix 是所有 key 的前缀，用于多应用共享同一 Redis 时隔离。
	Prefix string
	// Client 是装配期注入的底层客户端（如 Redis）。用 any 而不是具体类型，
	// 使能力核心不依赖任何驱动的 SDK；驱动自己做类型断言并给出明确错误。
	Client any
}

// drivers 是本构建已注册的驱动表。写入只发生在包初始化期（Go 保证 init 串行且先于
// main），启动后只读，因此不加锁。
var drivers = map[string]Factory{}

// Register 注册缓存驱动，仅供驱动包在 init() 中调用。重复注册是编码错误，直接 panic
// （与 database/sql.Register 同形）。
func Register(name string, f Factory) {
	if _, dup := drivers[name]; dup {
		panic("cache: driver already registered: " + name)
	}
	drivers[name] = f
}

// RegisteredDrivers 返回本构建已注册的驱动名（升序），用于 fail-closed 文案与诊断。
func RegisteredDrivers() []string {
	out := make([]string, 0, len(drivers))
	for name := range drivers {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// New 按驱动名创建缓存实例。驱动未编译进本构建时明确报错，不静默回退。
func New(driver string, opts Options) (Cache, error) {
	f, ok := drivers[driver]
	if !ok {
		return nil, fmt.Errorf("cache driver %q is not compiled into this build (compiled: %s)",
			driver, driversList(RegisteredDrivers()))
	}
	return f(opts)
}

func driversList(names []string) string {
	if len(names) == 0 {
		return "none"
	}
	return strings.Join(names, ", ")
}
