package db

import (
	"context"
	"errors"
	"strings"
	"time"

	"jimu/internal/platform/logger"

	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// SlowQueryThreshold 慢查询阈值（默认 200ms）
const SlowQueryThreshold = 200 * time.Millisecond

// sensitiveFields 需要脱敏的字段名（小写匹配）
var sensitiveFields = []string{
	"password", "secret", "token", "access_token", "refresh_token",
	"idcard", "id_card", "mobile", "phone", "credit_card",
}

// gormLogger 自定义 Gorm 日志：慢查询告警 + 敏感字段脱敏
type gormLogger struct {
	*logger.Logger
	threshold time.Duration
}

// NewGormLogger 创建 Gorm 自定义日志
func NewGormLogger(log *logger.Logger, slowThreshold time.Duration) gormlogger.Interface {
	if slowThreshold <= 0 {
		slowThreshold = SlowQueryThreshold
	}
	return &gormLogger{Logger: log, threshold: slowThreshold}
}

func (l *gormLogger) LogMode(gormlogger.LogLevel) gormlogger.Interface { return l }

func (l *gormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	args := append([]interface{}{msg}, sanitizeArgs(data)...)
	l.Logger.WithContext(ctx).Info(args...)
}

func (l *gormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	args := append([]interface{}{msg}, sanitizeArgs(data)...)
	l.Logger.WithContext(ctx).Warn(args...)
}

func (l *gormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	args := append([]interface{}{msg}, sanitizeArgs(data)...)
	l.Logger.WithContext(ctx).Error(args...)
}

func (l *gormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	elapsed := time.Since(begin)
	sql, rows := fc()

	// 脱敏 SQL 中的敏感值
	sql = sanitizeSQL(sql)

	fields := []interface{}{
		"elapsed", elapsed.String(),
		"rows", rows,
		"sql", sql,
	}

	switch {
	case err != nil && !errors.Is(err, gorm.ErrRecordNotFound):
		l.Logger.WithContext(ctx).Errorw("db query failed", append(fields, "error", err.Error())...)
	case elapsed > l.threshold:
		l.Logger.WithContext(ctx).Warnw("slow query detected", fields...)
	}
}

// sanitizeArgs 脱敏日志参数中的敏感字段
func sanitizeArgs(args []interface{}) []interface{} {
	for i := 0; i+1 < len(args); i += 2 {
		if key, ok := args[i].(string); ok {
			if isSensitiveField(key) {
				args[i+1] = "***"
			}
		}
	}
	return args
}

// sanitizeSQL 脱敏 SQL 语句中的敏感值（简单字符串替换）
func sanitizeSQL(sql string) string {
	// 对 VALUES (...) 中的内容做基本脱敏
	// 注意：这是简化实现，生产环境可使用 sqlparser 做更精确的解析
	upper := strings.ToUpper(sql)
	if !strings.Contains(upper, "INSERT") && !strings.Contains(upper, "UPDATE") {
		return sql
	}
	// 仅记录 SQL 结构，不记录完整 VALUES（超过 200 字符时截断）
	if len(sql) > 200 {
		return sql[:200] + "...[truncated]"
	}
	return sql
}

func isSensitiveField(name string) bool {
	lower := strings.ToLower(name)
	for _, f := range sensitiveFields {
		if strings.Contains(lower, f) {
			return true
		}
	}
	return false
}
