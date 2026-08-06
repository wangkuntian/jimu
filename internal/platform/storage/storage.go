package storage

import (
	"context"
	"io"
	"time"
)

// Storage 文件存储接口
// 支持本地、S3、OSS、MinIO 等多种实现
type Storage interface {
	// Upload 上传文件
	// key: 文件存储路径（如 "avatars/2024/01/abc.jpg"）
	// reader: 文件内容
	// size: 文件大小（字节）
	// contentType: MIME 类型
	Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error

	// Download 下载文件
	Download(ctx context.Context, key string) (io.ReadCloser, error)

	// Delete 删除文件
	Delete(ctx context.Context, key string) error

	// Exists 检查文件是否存在
	Exists(ctx context.Context, key string) (bool, error)

	// Size 获取文件大小
	Size(ctx context.Context, key string) (int64, error)

	// URL 获取文件访问 URL（公开访问）
	URL(key string) string

	// PresignedURL 获取预签名 URL（临时访问）
	// expiry: URL 有效期
	PresignedURL(key string, expiry time.Duration) (string, error)

	// PresignedUploadURL 获取预签名上传 URL
	// expiry: URL 有效期
	// contentType: 上传文件的 MIME 类型
	PresignedUploadURL(key string, expiry time.Duration, contentType string) (string, error)
}

// FileInfo 文件信息
type FileInfo struct {
	Key         string    `json:"key"`
	Size        int64     `json:"size"`
	ContentType string    `json:"content_type"`
	URL         string    `json:"url"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// UploadOptions 上传选项
type UploadOptions struct {
	ContentType string            // MIME 类型
	Metadata    map[string]string // 自定义元数据
	// 图片处理
	Width  int // 指定宽度（图片缩放）
	Height int // 指定高度（图片缩放）
}

// ListOptions 列表选项
type ListOptions struct {
	Prefix    string // 前缀过滤
	Delimiter string // 分隔符（模拟目录）
	Limit     int    // 最大返回数量
	Cursor    string // 分页游标
}

// Lister 文件列表接口（可选实现）
type Lister interface {
	List(ctx context.Context, opts ListOptions) ([]FileInfo, string, error)
}
