# Go Limiter

实现三种常用的限流器

1. 滑动窗口
2. 漏桶限流
3. 令牌池

## 滑动窗口

滑动窗口算法指有一个固定大小的时间窗口`T` ，设置在这个`T`窗口下只能通过`M`个请求，并且这个窗口是不断向前移动

计算公式 `remaining = M - (resetInSec / T * prevHit + currHit)`

resetInSec = exp - ts - time until end of current window.

* exp: 当前窗口的 end time
* ts: 当前时间

```text                
        prev                       curr
|----------------------|------------------------|
                                 ^              ^
                                 ts------------exp
                                    resetInSec
```

对于设置 exp

```go
// 设置下一个窗口，这里有两种可能
// 1. elapsed 超过窗口，那么直接创建一个新的窗口
// 2. 没有超过一个新窗口，s.exp + s.expiration
elapsed := ts - s.exp
if elapsed >= s.expiration {
    s.exp = ts + s.expiration
} else {
    s.exp = s.expiration + s.exp
}

```

## 漏桶限流

漏桶限流算法指有一个固定大小的漏桶以固定速率流出水，如果漏桶中没有水就不流出（相当于没有请求），如果流通的水过多就把多余的水丢弃（相当于请求过多）。

实现方法

1. 当请求 1 处理结束后, 我们记录下请求 1 的处理完成的时刻, 记为 limiter.last。
2. 当请求2到来，如果此刻的时间与 limiter.last 相比并没有达到 perRequest 的间隔大小，那么 sleep
   一段时间即可。`sleepFor = per - now.Sub(old.last) > 0`

```go
sleepFor = t.perRequest - now.Sub(t.last)
if sleepFor > 0 {
    t.clock.Sleep(sleepFor)
    t.last = now.Add(sleepFor)
} else {
    t.last = now
}
```

最大松弛量 （maxStack）

请求 1 完成后，15ms 后，请求 2 才到来，可以对请求 2 立即处理。请求 2 完成后，5ms 后，请求 3 到来，这个时候距离上次请求还不足
10ms，因此还需要等待 5ms。

```go
// 请求1执行后，15ms，请求2到达，此时sleepFor=-5ms，即有5ms的松弛时间
t.sleepFor += t.perRequest - now.Sub(t.last)
if t.sleepFor > 0 {
    t.clock.Sleep(t.sleepFor)
    t.last = now.Add(t.sleepFor)
    t.sleepFor = 0
} else {
    t.last = now
}
```

## 令牌池

令牌桶限流算法指有一个固定大小的桶并且按照固定速率向其放入令牌，当桶中令牌个数满了后就拒绝新添加的令牌。当一个请求过来时会去桶里拿一个令牌，如果拿到令牌就说明请求可以通过，如果没有拿到令牌就拒绝请求。

实现方法

使用`time.Ticker` 来设置规定时间向令牌桶送令牌

速率计算公式 `period/max` period 单位时间内，最大可以有多少令牌

---

# 参考资料k

[限流算法之一](https://hedzr.com/golang/algorithm/rate-limit-1/)

[uber-go 漏桶限流器使用与原理分析
](https://www.cyhone.com/articles/analysis-of-uber-go-ratelimit/)