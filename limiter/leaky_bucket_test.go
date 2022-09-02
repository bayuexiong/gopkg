package limiter

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestLeakyBucket_Take(t *testing.T) {
	rl := NewLeakyBucket(100) // per second

	prev := time.Now()
	for i := 0; i < 10; i++ {
		now := rl.Take()
		fmt.Println(i, now.Sub(prev))
		prev = now
	}
}

func TestLeakyBucket_Take_Concurrency(t *testing.T) {
	rl := NewLeakyBucket(100) // per second

	wg := sync.WaitGroup{}
	prev := time.Now()
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			now := rl.Take()
			fmt.Println(i, now.Sub(prev))
			prev = now
		}(i)
	}
	wg.Wait()
}
