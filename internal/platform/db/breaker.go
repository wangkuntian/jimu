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
	"jimu/internal/platform/logger"

	"gorm.io/gorm"
)

// breakerConnPool 在语句级别接入熔断：DB 不可用时快速失败，避免每个请求都等连接/查询超时。
// 只把连接/网络类错误计为失败（见 isDBTransportError），SQL 业务错误不影响熔断状态。
type breakerConnPool struct {
	pool *sql.DB
	b    *breaker.Breaker
}

// newBreakerConnPool 包装 sql.DB，实现 gorm.ConnPool 与 gorm.ConnPoolBeginner
func newBreakerConnPool(pool *sql.DB, b *breaker.Breaker) gorm.ConnPool {
	return &breakerConnPool{pool: pool, b: b}
}

// DB 保证 gorm.DB.DB() 仍能拿到底层连接池（健康检查、连接池指标依赖）
func (p *breakerConnPool) DB() *sql.DB { return p.pool }

func (p *breakerConnPool) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	if err := p.b.Allow(); err != nil {
		return nil, err
	}
	stmt, err := p.pool.PrepareContext(ctx, query)
	p.record(err)
	return stmt, err
}

func (p *breakerConnPool) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	if err := p.b.Allow(); err != nil {
		return nil, err
	}
	res, err := p.pool.ExecContext(ctx, query, args...)
	p.record(err)
	return res, err
}

func (p *breakerConnPool) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	if err := p.b.Allow(); err != nil {
		return nil, err
	}
	rows, err := p.pool.QueryContext(ctx, query, args...)
	p.record(err)
	return rows, err
}

// QueryRowContext 无法在此观察错误（错误在 Scan 时暴露）。熔断开启时用已取消的
// context 快速失败，避免真正发起查询；错误表现为 context.Canceled。
func (p *breakerConnPool) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	if err := p.b.Allow(); err != nil {
		cancelled, cancel := context.WithCancel(ctx)
		cancel()
		return p.pool.QueryRowContext(cancelled, query, args...)
	}
	return p.pool.QueryRowContext(ctx, query, args...)
}

func (p *breakerConnPool) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	if err := p.b.Allow(); err != nil {
		return nil, err
	}
	tx, err := p.pool.BeginTx(ctx, opts)
	p.record(err)
	return tx, err
}

func (p *breakerConnPool) record(err error) {
	if isDBTransportError(err) {
		p.b.OnFailure()
		return
	}
	p.b.OnSuccess()
}

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

// attachBreaker 在语句级挂载熔断。
// 注意：启用读写分离时 dbresolver 会为每个主/从库建立独立连接池并接管 ConnPool，
// 语句级熔断无法覆盖，此时跳过并给出提示。
func attachBreaker(db *gorm.DB, cfg config.BreakerConfig, split bool, log *logger.Logger) error {
	if !cfg.Enabled {
		return nil
	}
	if split {
		if log != nil {
			log.Warnw("db breaker skipped: read/write splitting enabled", "reason", "dbresolver manages its own conn pools")
		}
		return nil
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("get sql.DB for breaker: %w", err)
	}
	db.ConnPool = newBreakerConnPool(sqlDB, breaker.New("db", breaker.Config{
		MaxFailures:  cfg.MaxFailures,
		ResetTimeout: time.Duration(cfg.ResetTimeoutSec) * time.Second,
	}))
	return nil
}
