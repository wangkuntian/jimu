// Package authstore 定义 auth 能力的存储契约与驱动注册表。
//
// 能力核心只依赖本包的接口；redis/memory 驱动在 init() 中注册工厂。
// 这样 auth 核心不 import 任何 Redis SDK，驱动是否编译进构建由 profile 决定。
package authstore

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"jimu/internal/platform/identity"
)

// ResetCodeStore 是密码重置验证码的一次性存储。
type ResetCodeStore interface {
	Set(ctx context.Context, emailHash, code string) error
	GetAndDelete(ctx context.Context, emailHash string) (string, error)
	TTL() time.Duration
}

// Stores 是 auth 能力运行所需的全部存储实现。
type Stores struct {
	Sessions    identity.SessionStore
	RateCounter identity.RateCounter
	Failures    identity.FailureStore
	ResetCodes  ResetCodeStore
}

// Options 是驱动构造参数。
type Options struct {
	// RedisClient 是装配期注入的 Redis 客户端；memory 驱动忽略它。
	RedisClient any
	// ResetCodeTTL 是密码重置验证码有效期。
	ResetCodeTTL time.Duration
}

// Factory 构造一组 auth 存储实现。
type Factory func(Options) (Stores, error)

var drivers = map[string]Factory{}

// Register 注册 auth 存储驱动，仅供驱动包在 init() 中调用。
func Register(name string, f Factory) {
	if _, dup := drivers[name]; dup {
		panic("authstore: driver already registered: " + name)
	}
	drivers[name] = f
}

// RegisteredDrivers 返回本构建已注册的驱动名（升序）。
func RegisteredDrivers() []string {
	out := make([]string, 0, len(drivers))
	for name := range drivers {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// DefaultDriver 返回未显式配置时的驱动：优先 redis，其次 memory。
func DefaultDriver() string {
	if _, ok := drivers["redis"]; ok {
		return "redis"
	}
	if _, ok := drivers["memory"]; ok {
		return "memory"
	}
	return ""
}

// New 按驱动名构造存储。驱动未编译进本构建时明确报错。
func New(name string, opts Options) (Stores, error) {
	f, ok := drivers[name]
	if !ok {
		return Stores{}, fmt.Errorf("auth driver %q is not compiled into this build (compiled: %s)",
			name, driversList(RegisteredDrivers()))
	}
	return f(opts)
}

func driversList(names []string) string {
	if len(names) == 0 {
		return "none"
	}
	return strings.Join(names, ", ")
}
