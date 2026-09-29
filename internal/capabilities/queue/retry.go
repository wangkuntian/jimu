package queue

import "time"

// RetryStrategy 重试策略接口
type RetryStrategy interface {
	// NextDelay 返回下次重试的延迟时间，ok=false 表示不再重试
	NextDelay(attempt int) (delay time.Duration, ok bool)
}

// FixedRetry 固定间隔重试
type FixedRetry struct {
	Delay time.Duration
}

func (r FixedRetry) NextDelay(attempt int) (time.Duration, bool) {
	return r.Delay, true
}

// ExponentialBackoff 指数退避重试
type ExponentialBackoff struct {
	BaseDelay time.Duration
	MaxDelay  time.Duration
}

func (r ExponentialBackoff) NextDelay(attempt int) (time.Duration, bool) {
	delay := r.BaseDelay << attempt // 2^attempt * base
	if delay > r.MaxDelay {
		delay = r.MaxDelay
	}
	return delay, true
}

// DefaultRetry 默认重试策略（指数退避）
var DefaultRetry = ExponentialBackoff{
	BaseDelay: time.Second,
	MaxDelay:  60 * time.Second,
}
