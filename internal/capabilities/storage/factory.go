package storage

import (
	"fmt"
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

// New 创建存储实例
func New(cfg Config) (Storage, error) {
	switch cfg.Type {
	case StorageTypeLocal, "":
		if cfg.BaseDir == "" {
			cfg.BaseDir = "storage"
		}
		if cfg.BaseURL == "" {
			cfg.BaseURL = "/files"
		}
		return NewLocalStorage(cfg.BaseDir, cfg.BaseURL)
	case StorageTypeS3:
		return newS3Storage(cfg)
	case StorageTypeOSS:
		return newOSSStorage(cfg)
	case StorageTypeMinIO:
		return newMinioStorage(cfg)
	default:
		return nil, fmt.Errorf("unsupported storage type: %s", cfg.Type)
	}
}

// newOSSStorage 创建阿里云 OSS 存储。
// OSS 兼容 S3 协议，复用 S3 SDK（path style + endpoint），无需引入 aliyun-oss-go-sdk。
func newOSSStorage(cfg Config) (Storage, error) {
	return newS3CompatibleStorage(cfg, false)
}
