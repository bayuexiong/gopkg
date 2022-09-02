// Package limiter
// 参考：
// https://github.com/kevinms/leakybucket-go
// https://github.com/uber-go/ratelimit/
// https://www.cyhone.com/articles/analysis-of-uber-go-ratelimit/
// https://hedzr.com/golang/algorithm/rate-limit-1/#%E6%BC%8F%E6%A1%B6%E6%B3%95leaky-bucket
package limiter

import (
	"sync/atomic"
	"time"
	"unsafe"
)

type LeakyBucket struct {
	state      unsafe.Pointer
	perRequest time.Duration // 多长时间可以执行
	maxSlack   time.Duration
}

type state struct {
	last     time.Time
	sleepFor time.Duration
}

func NewLeakyBucket(rate int) *LeakyBucket {
	l := &LeakyBucket{
		perRequest: time.Second / time.Duration(rate),
		maxSlack:   -1 * 10 * time.Duration(rate),
	}
	initialState := state{
		last:     time.Time{},
		sleepFor: 0,
	}
	atomic.StorePointer(&l.state, unsafe.Pointer(&initialState))
	return l
}

func (b *LeakyBucket) Take() time.Time {
	var (
		newState state
		taken    bool
		interval time.Duration
	)
	for !taken {
		now := time.Now()

		previousStatePointer := atomic.LoadPointer(&b.state)
		oldState := (*state)(previousStatePointer)

		newState = state{
			last:     now,
			sleepFor: oldState.sleepFor,
		}

		// 第一次使用
		if oldState.last.IsZero() {
			taken = atomic.CompareAndSwapPointer(&b.state, previousStatePointer, unsafe.Pointer(&newState))
			continue
		}

		// 请求1执行后，15ms，请求2到达，此时sleepFor=-5ms，即对于后面的请求有5ms的松弛时间
		// 请求3，5ms之内到达，即 10ms - 5ms - 5ms，请求3可以直接执行
		newState.sleepFor += b.perRequest - now.Sub(oldState.last)
		// 避免请求2在非常久的时间后到达，给了一个特别大的松弛时间
		if newState.sleepFor < b.maxSlack {
			newState.sleepFor = b.maxSlack
		}

		// 需要等待多久，需要清空sleepFor避免影响松弛量
		if newState.sleepFor > 0 {
			newState.last = newState.last.Add(newState.sleepFor)
			interval, newState.sleepFor = newState.sleepFor, 0
		}
		taken = atomic.CompareAndSwapPointer(&b.state, previousStatePointer, unsafe.Pointer(&newState))
	}
	time.Sleep(interval)
	return newState.last
}
