package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// LocalStorage 本地文件存储实现
type LocalStorage struct {
	baseDir string // 存储根目录
	baseURL string // 访问 URL 前缀
}

// NewLocalStorage 创建本地存储
func NewLocalStorage(baseDir, baseURL string) (*LocalStorage, error) {
	// 确保目录存在
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, fmt.Errorf("create storage directory: %w", err)
	}

	return &LocalStorage{
		baseDir: baseDir,
		baseURL: strings.TrimSuffix(baseURL, "/"),
	}, nil
}

func (s *LocalStorage) Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error {
	fullPath := s.fullPath(key)

	// 确保父目录存在
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	// 创建临时文件，然后原子重命名
	tmpFile := fullPath + ".tmp"
	f, err := os.Create(tmpFile)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}

	written, err := io.Copy(f, reader)
	if closeErr := f.Close(); closeErr != nil && err == nil {
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(tmpFile)
		return fmt.Errorf("write file: %w", err)
	}

	if size > 0 && written != size {
		_ = os.Remove(tmpFile)
		return fmt.Errorf("size mismatch: expected %d, got %d", size, written)
	}

	// 原子重命名
	if err := os.Rename(tmpFile, fullPath); err != nil {
		_ = os.Remove(tmpFile)
		return fmt.Errorf("rename file: %w", err)
	}

	return nil
}

func (s *LocalStorage) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	fullPath := s.fullPath(key)
	f, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", key)
		}
		return nil, fmt.Errorf("open file: %w", err)
	}
	return f, nil
}

func (s *LocalStorage) Delete(ctx context.Context, key string) error {
	fullPath := s.fullPath(key)
	if err := os.Remove(fullPath); err != nil {
		if os.IsNotExist(err) {
			return nil // 不存在视为删除成功
		}
		return fmt.Errorf("delete file: %w", err)
	}
	return nil
}

func (s *LocalStorage) Exists(ctx context.Context, key string) (bool, error) {
	fullPath := s.fullPath(key)
	_, err := os.Stat(fullPath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func (s *LocalStorage) Size(ctx context.Context, key string) (int64, error) {
	fullPath := s.fullPath(key)
	info, err := os.Stat(fullPath)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

func (s *LocalStorage) URL(key string) string {
	return s.baseURL + "/" + key
}

func (s *LocalStorage) PresignedURL(key string, expiry time.Duration) (string, error) {
	// 本地存储不支持预签名，返回普通 URL
	return s.URL(key), nil
}

func (s *LocalStorage) PresignedUploadURL(key string, expiry time.Duration, contentType string) (string, error) {
	// 本地存储不支持预签名上传，返回空
	return "", fmt.Errorf("local storage does not support presigned upload")
}

func (s *LocalStorage) fullPath(key string) string {
	// 防止路径遍历攻击
	key = strings.TrimPrefix(key, "/")
	key = strings.ReplaceAll(key, "..", "")
	return filepath.Join(s.baseDir, key)
}

var _ Storage = (*LocalStorage)(nil)
