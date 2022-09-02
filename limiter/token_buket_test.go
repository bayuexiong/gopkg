package limiter

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestNewTokenBucket(t *testing.T) {
	tb := NewTokenBucket(10)
	tb.Start(time.Millisecond * 200)
	time.Sleep(time.Second)
	assert.True(t, tb.Taken(5))
	assert.False(t, tb.Taken(1))
	time.Sleep(time.Millisecond * 200)
	assert.True(t, tb.Taken(1))

}
