package storage

import (
	"fmt"
	"sort"
	"strings"
)

// StorageType 存储类型
type StorageType string

const (
	StorageTypeLocal StorageType = "local"
	StorageTypeS3    StorageType = "s3"
	StorageTypeOSS   StorageType = "oss"
	StorageTypeMinIO StorageType = "minio"
)

// Config 存储配置
type Config struct {
	Type StorageType `mapstructure:"type"`

	// 本地存储
	BaseDir string `mapstructure:"base_dir"`
	BaseURL string `mapstructure:"base_url"`

	// S3/OSS/MinIO 通用
	Endpoint  string `mapstructure:"endpoint"`
	Region    string `mapstructure:"region"`
	Bucket    string `mapstructure:"bucket"`
	AccessKey string `mapstructure:"access_key"`
	SecretKey string `mapstructure:"secret_key"`
	// 是否使用路径风格（MinIO 需要）
	PathStyle bool `mapstructure:"path_style"`
}

// Factory 按配置构造存储实现。驱动包在 init() 中调用 Register 注册。
type Factory func(Config) (Storage, error)

// drivers 是本构建已注册的驱动表。写入只发生在包初始化期（Go 保证 init 串行且先于
// main），启动后只读，因此不加锁。
var drivers = map[StorageType]Factory{}

// Register 注册存储驱动，仅供驱动包在 init() 中调用。重复注册是编码错误，直接 panic
// （与 database/sql.Register 同形）。
func Register(t StorageType, f Factory) {
	if _, dup := drivers[t]; dup {
		panic("storage: driver already registered: " + string(t))
	}
	drivers[t] = f
}

// RegisteredTypes 返回本构建已注册的存储类型（升序），用于 fail-closed 文案与诊断。
func RegisteredTypes() []StorageType {
	out := make([]StorageType, 0, len(drivers))
	for t := range drivers {
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// New 按配置创建存储实例：空类型按 local 处理（与下沉前一致）。配置的类型未编译进
// 本构建时明确报错，不静默回退到其它驱动。
func New(cfg Config) (Storage, error) {
	if cfg.Type == "" {
		cfg.Type = StorageTypeLocal
	}
	f, ok := drivers[cfg.Type]
	if !ok {
		return nil, fmt.Errorf("storage driver %q is not compiled into this build (compiled: %s)",
			cfg.Type, typesList(RegisteredTypes()))
	}
	return f(cfg)
}

// typesList 渲染已注册类型清单；空集渲染为 none。
func typesList(types []StorageType) string {
	if len(types) == 0 {
		return "none"
	}
	out := make([]string, len(types))
	for i, t := range types {
		out[i] = string(t)
	}
	return strings.Join(out, ", ")
}
