// Package breaker 提供可复用的进程内熔断器：连续失败达到阈值后开启，
// 冷却期内快速失败，冷却结束进入半开态放行单个探测请求，探测成功则恢复。
// 供 Redis/DB 等外部依赖复用（HTTP 出站客户端另有独立实现，见 kernel/httpclient）。
package breaker

import (
	"errors"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// ErrOpen 熔断开启时返回，调用方应快速失败（不要重试）
var ErrOpen = errors.New("breaker: circuit open")

// State 熔断状态
type State int

const (
	Closed State = iota
	Open
	HalfOpen
)

const (
	defaultMaxFailures  = 5
	defaultResetTimeout = 10 * time.Second
)

var (
	openGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "jimu",
		Subsystem: "breaker",
		Name:      "open",
		Help:      "Whether the circuit breaker is open (1) or closed (0)",
	}, []string{"component"})

	rejectedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "jimu",
		Subsystem: "breaker",
		Name:      "rejected_total",
		Help:      "Total number of calls rejected by an open circuit breaker",
	}, []string{"component"})

	tripTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "jimu",
		Subsystem: "breaker",
		Name:      "trip_total",
		Help:      "Total number of times the circuit breaker opened",
	}, []string{"component"})
)

// Config 熔断配置
type Config struct {
	MaxFailures  int           // 连续失败阈值，<=0 使用默认 5
	ResetTimeout time.Duration // 冷却时间，<=0 使用默认 10s
}

// Breaker 并发安全的熔断器
type Breaker struct {
	component    string
	maxFailures  int
	resetTimeout time.Duration

	mu            sync.Mutex
	state         State
	consecutive   int
	openedAt      time.Time
	probeInFlight bool
}

// New 创建熔断器；component 用于指标标签（如 redis/db）
func New(component string, cfg Config) *Breaker {
	if cfg.MaxFailures <= 0 {
		cfg.MaxFailures = defaultMaxFailures
	}
	if cfg.ResetTimeout <= 0 {
		cfg.ResetTimeout = defaultResetTimeout
	}
	openGauge.WithLabelValues(component).Set(0)
	return &Breaker{
		component:    component,
		maxFailures:  cfg.MaxFailures,
		resetTimeout: cfg.ResetTimeout,
		state:        Closed,
	}
}

// Allow 判断是否允许调用；返回 ErrOpen 时应快速失败。
// 冷却结束后的首个调用会获得一次半开探测机会。
func (b *Breaker) Allow() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch b.state {
	case Closed:
		return nil
	case Open:
		if time.Since(b.openedAt) >= b.resetTimeout {
			b.state = HalfOpen
			b.probeInFlight = true
			return nil
		}
		rejectedTotal.WithLabelValues(b.component).Inc()
		return ErrOpen
	default: // HalfOpen：只放行一个在途探测
		if b.probeInFlight {
			rejectedTotal.WithLabelValues(b.component).Inc()
			return ErrOpen
		}
		b.probeInFlight = true
		return nil
	}
}

// OnSuccess 上报一次成功调用
func (b *Breaker) OnSuccess() { b.record(true) }

// OnFailure 上报一次失败调用（仅传输类失败应上报，业务错误不算）
func (b *Breaker) OnFailure() { b.record(false) }

// State 返回当前状态
func (b *Breaker) State() State {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.state
}

func (b *Breaker) record(success bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch b.state {
	case Closed:
		if success {
			b.consecutive = 0
			return
		}
		b.consecutive++
		if b.consecutive >= b.maxFailures {
			b.tripLocked()
		}
	case HalfOpen:
		b.probeInFlight = false
		if success {
			b.state = Closed
			b.consecutive = 0
			openGauge.WithLabelValues(b.component).Set(0)
			return
		}
		b.tripLocked()
	case Open:
		// 冷却期内的结果不重复计数
	}
}

// tripLocked 进入开启态（调用方持锁）
func (b *Breaker) tripLocked() {
	b.state = Open
	b.openedAt = time.Now()
	b.consecutive = 0
	b.probeInFlight = false
	openGauge.WithLabelValues(b.component).Set(1)
	tripTotal.WithLabelValues(b.component).Inc()
}
