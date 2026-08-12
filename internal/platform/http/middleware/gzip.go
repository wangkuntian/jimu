package middleware

import (
	"compress/gzip"
	"strings"

	"github.com/gin-gonic/gin"
)

// gzipResponseWriter 包装 gin.ResponseWriter 以支持 gzip 写入
type gzipResponseWriter struct {
	gin.ResponseWriter
	writer *gzip.Writer
}

func (g *gzipResponseWriter) Write(data []byte) (int, error) {
	return g.writer.Write(data)
}

func (g *gzipResponseWriter) WriteString(s string) (int, error) {
	return g.writer.Write([]byte(s))
}

// GzipCompression gzip 响应压缩中间件
// 仅当客户端 Accept-Encoding 包含 gzip 且响应体大于阈值时压缩
func GzipCompression() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查客户端是否支持 gzip
		if !strings.Contains(c.GetHeader("Accept-Encoding"), "gzip") {
			c.Next()
			return
		}

		// 跳过已压缩的内容类型
		contentType := c.ContentType()
		if isAlreadyCompressed(contentType) {
			c.Next()
			return
		}

		gz := gzip.NewWriter(c.Writer)
		defer func() { _ = gz.Close() }()

		gzw := &gzipResponseWriter{ResponseWriter: c.Writer, writer: gz}
		c.Writer = gzw
		c.Header("Content-Encoding", "gzip")
		c.Header("Vary", "Accept-Encoding")

		c.Next()
	}
}

// isAlreadyCompressed 判断内容类型是否已压缩（无需再次压缩）
func isAlreadyCompressed(contentType string) bool {
	compressedTypes := []string{
		"image/",
		"audio/",
		"video/",
		"application/gzip",
		"application/zip",
		"application/x-7z-compressed",
		"application/x-rar-compressed",
		"application/pdf",
	}
	for _, t := range compressedTypes {
		if strings.Contains(contentType, t) {
			return true
		}
	}
	return false
}
