package limiter

import (
	"sync"
	"testing"
	"time"
)

func Test_Sliding_Window_Limiter_Concurrency(t *testing.T) {
	sd := NewSlidingWindow(50, time.Minute)
	wg := sync.WaitGroup{}

	result := true

	for i := 0; i < 51; i++ {
		wg.Add(1)
		go func() {
			result = result && sd.Limit()
			wg.Done()
		}()
	}
	wg.Wait()
	if result {
		t.Error("sliding window limiter error")
	}
}

func Test_Sliding_Window_Concurrency(t *testing.T) {
	sd := NewSlidingWindow(50, time.Minute)
	wg := sync.WaitGroup{}

	result := true

	for i := 0; i < 49; i++ {
		wg.Add(1)
		go func() {
			result = result && sd.Limit()
			wg.Done()
		}()
	}
	wg.Wait()
	if !result {
		t.Error("Test_Sliding_Window_Concurrency error")
	}

}

func Test_Sliding_Window_Skip(t *testing.T) {
	sd := NewSlidingWindow(10, time.Second*3)

	result := true
	for i := 0; i < 20; i++ {
		result = result && sd.Limit()
	}
	if result == true {
		t.Error("sliding limit error")
	}
	time.Sleep(time.Second * 3)
	if sd.Limit() != false {
		t.Error("根据 prevHit 应该返回false")
	}
	time.Sleep(time.Second * 3)
	if sd.Limit() != true {
		t.Error("根据 prevHit 应该返回true")
	}

}
