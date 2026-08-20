package http

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"jimu/internal/platform/storage"
	"jimu/internal/shared/errors"
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// UploadHandler 通用文件上传处理器
type UploadHandler struct {
	storage    storage.Storage
	maxSize    int64  // 最大文件大小（字节）
	allowTypes string // 允许的 MIME 前缀，逗号分隔，空表示允许所有
	basePrefix string // 存储路径前缀，如 "uploads"
	scanner    Scanner // 可选安全扫描器，nil 表示不扫描
}

// UploadConfig 上传处理器配置
type UploadConfig struct {
	Storage    storage.Storage
	MaxSize    int64  // 默认 10MB
	AllowTypes string // 如 "image/,application/pdf"
	BasePrefix string // 默认 "uploads"
	Scanner    Scanner // 可选病毒扫描器，nil 表示不扫描
}

// NewUploadHandler 创建上传处理器
func NewUploadHandler(cfg UploadConfig) *UploadHandler {
	if cfg.MaxSize <= 0 {
		cfg.MaxSize = 10 * 1024 * 1024 // 10MB
	}
	if cfg.BasePrefix == "" {
		cfg.BasePrefix = "uploads"
	}
	return &UploadHandler{
		storage:    cfg.Storage,
		maxSize:    cfg.MaxSize,
		allowTypes: cfg.AllowTypes,
		basePrefix: cfg.BasePrefix,
		scanner:    cfg.Scanner,
	}
}

// UploadResponse 上传响应
type UploadResponse struct {
	Key         string `json:"key"`
	URL         string `json:"url"`
	Size        int64  `json:"size"`
	ContentType string `json:"content_type"`
}

// HandleUpload 处理单文件上传
// formField: 表单中文件字段名，默认 "file"
func (h *UploadHandler) HandleUpload(formField ...string) gin.HandlerFunc {
	field := "file"
	if len(formField) > 0 && formField[0] != "" {
		field = formField[0]
	}

	return func(c *gin.Context) {
		file, header, err := c.Request.FormFile(field)
		if err != nil {
			response.Fail(c, errors.New(errors.CodeInvalidParam, "file is required"))
			return
		}

		result, err := h.upload(c.Request.Context(), file, header)
		_ = file.Close()
		if err != nil {
			response.Fail(c, err)
			return
		}
		response.OK(c, result)
	}
}

// HandleUploadMultiple 处理多文件上传
func (h *UploadHandler) HandleUploadMultiple(formField string, maxFiles int) gin.HandlerFunc {
	if maxFiles <= 0 || maxFiles > 20 {
		maxFiles = 10
	}

	return func(c *gin.Context) {
		form, err := c.MultipartForm()
		if err != nil {
			response.Fail(c, errors.New(errors.CodeInvalidParam, "invalid multipart form"))
			return
		}
		files := form.File[formField]
		if len(files) == 0 {
			response.Fail(c, errors.New(errors.CodeInvalidParam, "no files uploaded"))
			return
		}
		if len(files) > maxFiles {
			response.Fail(c, errors.New(errors.CodeInvalidParam, fmt.Sprintf("too many files, max %d", maxFiles)))
			return
		}

		results := make([]UploadResponse, 0, len(files))
		for _, header := range files {
			f, openErr := header.Open()
			if openErr != nil {
				response.Fail(c, errors.New(errors.CodeInvalidParam, "failed to read file"))
				return
			}
			result, uploadErr := h.upload(c.Request.Context(), f, header)
			_ = f.Close()
			if uploadErr != nil {
				response.Fail(c, uploadErr)
				return
			}
			results = append(results, *result)
		}
		response.OK(c, results)
	}
}

// HandleDelete 处理文件删除
func (h *UploadHandler) HandleDelete() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.Query("key")
		if key == "" {
			response.Fail(c, errors.New(errors.CodeInvalidParam, "key is required"))
			return
		}
		if err := h.storage.Delete(c.Request.Context(), key); err != nil {
			response.Fail(c, errors.Wrap(errors.CodeInternalError, "failed to delete file", err))
			return
		}
		response.OK(c, gin.H{"deleted": key})
	}
}

func (h *UploadHandler) upload(ctx context.Context, file io.Reader, header *multipart.FileHeader) (*UploadResponse, error) {
	// 检查文件大小
	if header.Size > h.maxSize {
		return nil, errors.New(errors.CodeInvalidParam,
			fmt.Sprintf("file too large, max %d bytes", h.maxSize))
	}
	if header.Size == 0 {
		return nil, errors.New(errors.CodeInvalidParam, "file is empty")
	}

	// 检查类型。Content-Type 头可由客户端伪造，嗅探 magic byte 覆盖头声明。
	contentType := header.Header.Get("Content-Type")
	data, err := readAll(file, h.maxSize)
	if err != nil {
		return nil, errors.Wrap(errors.CodeInternalError, "failed to read file", err)
	}
	if sniffed := http.DetectContentType(data); sniffed != "" {
		contentType = sniffed
	}
	if h.allowTypes != "" && !isAllowedType(contentType, h.allowTypes) {
		return nil, errors.New(errors.CodeInvalidParam,
			fmt.Sprintf("file type %s not allowed", contentType))
	}

	// 生成存储路径
	ext := filepath.Ext(header.Filename)
	if ext == "" {
		ext = ".bin"
	}
	now := time.Now()
	key := fmt.Sprintf("%s/%d/%02d/%02d/%s%s",
		h.basePrefix, now.Year(), now.Month(), now.Day(),
		uuid.NewString(), ext)

	// 安全扫描：scanner 非 nil 时落库前扫描。
	// fail-closed：不干净或扫描不可达均拒绝落库，坏文件不持久化。
	if h.scanner != nil {
		clean, scanErr := h.scanner.Scan(ctx, bytes.NewReader(data))
		if scanErr != nil {
			return nil, errors.Wrap(errors.CodeInternalError, "file security scan unavailable", scanErr)
		}
		if !clean {
			return nil, errors.New(errors.CodeInvalidParam, "file failed security scan")
		}
	}

	// 上传到存储
	if err := h.storage.Upload(ctx, key, bytes.NewReader(data), header.Size, contentType); err != nil {
		return nil, errors.Wrap(errors.CodeInternalError, "failed to upload file", err)
	}

	return &UploadResponse{
		Key:         key,
		URL:         h.storage.URL(key),
		Size:        header.Size,
		ContentType: contentType,
	}, nil
}

func isAllowedType(contentType, allowTypes string) bool {
	for _, prefix := range strings.Split(allowTypes, ",") {
		if strings.HasPrefix(contentType, strings.TrimSpace(prefix)) {
			return true
		}
	}
	return false
}

func readAll(r io.Reader, max int64) ([]byte, error) {
	// 限制读取量，防止内存溢出
	limited := io.LimitReader(r, max+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > max {
		return nil, fmt.Errorf("file exceeds max size %d", max)
	}
	return data, nil
}
