// Package limiter
package limiter

import (
	"sync"
	"time"
)

type Limit float64

// Limiter time/rate 的简化版，实现一个lazy的令牌池
type Limiter struct {
	// 1s 生成多少个token
	limit  Limit
	bucket int

	mu    sync.Mutex
	token float64 // 桶中目前剩余的token数目，可以为负数。
	// last is the last time the limiter's tokens field was updated
	last time.Time
	// lastEvent is the latest time of a rate-limited event (past or future)
	lastEvent time.Time
}

// NewLimiter limit Limit 每秒产生多少token
func NewLimiter(limit Limit, bucket int) *Limiter {
	return &Limiter{
		limit:  limit,
		bucket: bucket,
	}
}

// NewLimiterDuration d time.Duration how much duration generate a token
func NewLimiterDuration(d time.Duration, bucket int) *Limiter {
	return &Limiter{
		limit:  Limit(1 / float64(d.Seconds())),
		bucket: bucket,
	}
}

// Reservation  申请的返回结构
type Reservation struct {
	ok        bool
	lim       *Limiter
	tokens    int
	timeToAct time.Time
	// This is the Limit at reservation time, it can change later.
	limit Limit
}

func (l *Limiter) Allow() bool {
	return l.AllowN(time.Now(), 1)
}

func (l *Limiter) AllowN(now time.Time, n int) bool {
	reserve := l.reserveN(now, n, 0)
	return reserve.ok
}

func (l *Limiter) Wait(now time.Time, maxFutureReserve time.Duration) bool {
	return l.WaitN(now, 1, maxFutureReserve)
}

func (l *Limiter) WaitN(now time.Time, n int, maxFutureReserve time.Duration) bool {
	reserve := l.reserveN(now, n, maxFutureReserve)
	if !reserve.ok {
		return false
	}
	delay := reserve.timeToAct.Sub(now)
	if delay <= 0 {
		return true
	}
	ticker := time.NewTicker(delay)

	<-ticker.C
	return true
}

// 1. 计算上一次last到现在产生多少token,判断是否超过bucket
// 2. 计算申请后剩余的token
// 3. 判断是否可以获取，n < bucket && waitDuration <= maxFutureReserve
// 4. 可以获取，更新token，更新last，更新lastEvent,返回reservation
func (l *Limiter) reserveN(now time.Time, n int, maxFutureReserve time.Duration) *Reservation {
	l.mu.Lock()
	defer l.mu.Unlock()

	// 获取新增的token，判断是否超过bucket
	delta := l.tokensFromDuration(now.Sub(l.last))
	tokens := l.token + delta
	if bucket := float64(l.bucket); tokens > bucket {
		tokens = bucket
	}

	reserve := &Reservation{
		lim:    l,
		tokens: n,
		limit:  l.limit,
	}

	remaining := tokens - float64(n)

	// 计算等待时间
	var waitDuration time.Duration
	if remaining < 0 {
		waitDuration = l.durationFromTokens(-remaining)
	}

	// 判断是否可以获取
	ok := n <= l.bucket && waitDuration <= maxFutureReserve

	reserve.ok = ok
	// 可以获取更新limit的数值
	if ok {
		timeToAct := now.Add(waitDuration)
		reserve.timeToAct = timeToAct
		l.last = now
		l.lastEvent = reserve.timeToAct
		l.token = remaining
	}

	return reserve

}

func (l *Limiter) tokensFromDuration(duration time.Duration) float64 {
	return duration.Seconds() * float64(l.limit)
}

func (l *Limiter) durationFromTokens(token float64) time.Duration {
	seconds := token / float64(l.limit)
	return time.Duration(float64(time.Second) * seconds)
}
