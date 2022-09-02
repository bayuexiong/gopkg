package limiter

import (
	"sync"
	"sync/atomic"
	"time"
)

type SlidingWindow struct {
	max        int
	curHit     int
	prevHit    int
	timestamp  uint64
	exp        uint64
	expiration uint64
	mux        sync.RWMutex
}

func NewSlidingWindow(max int, expiration time.Duration) *SlidingWindow {
	s := &SlidingWindow{
		max:        max,
		expiration: uint64(expiration.Seconds()),
	}
	s.loop()
	return s
}

func (s *SlidingWindow) loop() {
	go func() {
		for {
			atomic.StoreUint64(&s.timestamp, uint64(time.Now().Unix()))
			time.Sleep(1 * time.Second)
		}
	}()
}

func (s *SlidingWindow) Limit() bool {
	s.mux.Lock()
	defer s.mux.Unlock()

	ts := atomic.LoadUint64(&s.timestamp)
	if s.exp == 0 {
		s.exp = ts + s.expiration
	} else if ts >= s.exp {

		// ts > s.exp , 当前时间已经超过上一个窗口
		s.prevHit = s.curHit

		// 初始化当前计数器
		s.curHit = 0

		// 设置下一个窗口，这里有两种可能
		// 1. elapsed 超过窗口，那么直接创建一个新的窗口
		// 2. 没有超过一个新窗口，s.exp + s.expiration
		elapsed := ts - s.exp
		if elapsed >= s.expiration {
			s.exp = ts + s.expiration
		} else {
			s.exp = s.expiration + s.exp
		}
	}

	s.curHit++

	resetInSec := s.exp - ts

	weight := float64(resetInSec) / float64(s.expiration)

	rate := int(weight*float64(s.prevHit)) + s.curHit

	remaining := s.max - rate

	if remaining < 0 {
		return false
	}
	return true
}
