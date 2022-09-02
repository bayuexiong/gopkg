package limiter

import (
	"sync/atomic"
	"time"
)

// TokenBucket 不够完善的令牌池, 通过time.Ticker实现
type TokenBucket struct {
	bucket int64
	count  int64
	exitCh chan struct{}
}

func NewTokenBucket(bucket int64) *TokenBucket {
	return &TokenBucket{
		bucket: bucket,
		exitCh: make(chan struct{}),
	}
}

func (t *TokenBucket) Count() int64 { return atomic.LoadInt64(&t.count) }

// Start d time.Duration how much duration generate a token
func (t *TokenBucket) Start(d time.Duration) *TokenBucket {

	go t.loop(d)
	return t
}

func (t *TokenBucket) loop(d time.Duration) {
	ticker := time.NewTicker(d)
	defer ticker.Stop()

	for {
		select {
		case <-t.exitCh:
			return
		case <-ticker.C:
			v := atomic.AddInt64(&t.count, 1)
			if v < t.bucket {
				continue
			}
			// 重新计数
			v = v % t.bucket
			if v > 0 {
				atomic.StoreInt64(&t.count, t.bucket)
			}
		}
	}
}

func (t *TokenBucket) taken(count int) bool {
	if vn := atomic.AddInt64(&t.count, int64(-1*count)); vn >= 0 {
		return true
	}
	atomic.AddInt64(&t.count, int64(count))
	return false
}

func (t *TokenBucket) Taken(count int) bool {
	return t.taken(count)
}

func (t *TokenBucket) Limit() bool {
	return t.taken(1)
}

func (t *TokenBucket) Close() {
	close(t.exitCh)
}
