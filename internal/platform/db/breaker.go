package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"jimu/internal/config"
	"jimu/internal/platform/breaker"

	"gorm.io/gorm"
)

// isDBTransportError 判断是否为连接/网络类错误。
// 刻意不把 context.DeadlineExceeded 计入：慢查询超时未必代表数据库不可用，
// 计入会导致误熔断（而连接不可用通常表现为 refused/reset/bad conn 等）。
func isDBTransportError(err error) bool {
	if err == nil {
		return false
	}
	// 先排除 context 错误：context.DeadlineExceeded 也实现了 net.Error，
	// 但慢查询超时/客户端取消不代表数据库不可用
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return false
	}
	if errors.Is(err, driver.ErrBadConn) ||
		errors.Is(err, sql.ErrConnDone) ||
		errors.Is(err, io.EOF) ||
		errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	msg := strings.ToLower(err.Error())
	for _, s := range []string{
		"connection refused", "connection reset", "broken pipe",
		"i/o timeout", "no such host", "server has gone away",
		"too many connections", "bad connection", "connection is closed",
		"unexpected eof", "can't connect to",
	} {
		if strings.Contains(msg, s) {
			return true
		}
	}
	return false
}

// attachBreaker 在 gorm 回调层挂载语句级熔断：语句执行前 Allow，执行后按错误类型记成功/失败。
// 走回调而不是包装 ConnPool，因此读写分离（dbresolver 为每个主/从库建独立连接池）同样生效，
// 且不会影响 gorm.DB()、连接池上限等对底层 *sql.DB 的操作。
func attachBreaker(db *gorm.DB, cfg config.BreakerConfig) error {
	if !cfg.Enabled {
		return nil
	}
	plugin := &breakerPlugin{b: breaker.New("db", breaker.Config{
		MaxFailures:  cfg.MaxFailures,
		ResetTimeout: time.Duration(cfg.ResetTimeoutSec) * time.Second,
	})}
	if err := db.Use(plugin); err != nil {
		return fmt.Errorf("register db breaker: %w", err)
	}
	return nil
}

// breakerPlugin 以 gorm 插件形式在每种语句处理器的首尾挂载熔断。
type breakerPlugin struct {
	b *breaker.Breaker
}

func (p *breakerPlugin) Name() string { return "jimu:db-breaker" }

// Initialize 为 Create/Query/Update/Delete/Row/Raw 六类操作注册 allow/record 回调。
// Before("*")/After("*") 保证在处理器内所有回调之外执行。
func (p *breakerPlugin) Initialize(db *gorm.DB) error {
	create, query := db.Callback().Create(), db.Callback().Query()
	update, remove := db.Callback().Update(), db.Callback().Delete()
	row, raw := db.Callback().Row(), db.Callback().Raw()

	type callback interface {
		Register(string, func(*gorm.DB)) error
	}
	processors := []struct {
		op            string
		before, after callback
	}{
		{"create", create.Before("*"), create.After("*")},
		{"query", query.Before("*"), query.After("*")},
		{"update", update.Before("*"), update.After("*")},
		{"delete", remove.Before("*"), remove.After("*")},
		{"row", row.Before("*"), row.After("*")},
		{"raw", raw.Before("*"), raw.After("*")},
	}
	for _, item := range processors {
		if err := item.before.Register("jimu:breaker:allow:"+item.op, p.allow); err != nil {
			return err
		}
		if err := item.after.Register("jimu:breaker:record:"+item.op, p.record); err != nil {
			return err
		}
	}
	return nil
}

// allow 熔断开启时把错误写入语句，后续回调因 db.Error 非空而不会真正访问数据库。
func (p *breakerPlugin) allow(db *gorm.DB) {
	if db.Error != nil {
		return
	}
	if err := p.b.Allow(); err != nil {
		// 错误写入语句即中止后续回调（gorm 的 SQL 回调都以 db.Error == nil 为前提）
		_ = db.AddError(err)
	}
}

// record 结算本次语句：熔断拒绝已计入 rejected，不重复计数。
func (p *breakerPlugin) record(db *gorm.DB) {
	if errors.Is(db.Error, breaker.ErrOpen) {
		return
	}
	if isDBTransportError(db.Error) {
		p.b.OnFailure()
		return
	}
	p.b.OnSuccess()
}
