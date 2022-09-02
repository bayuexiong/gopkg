package limiter

import (
	"testing"
	"time"
)

const (
	d = 100 * time.Millisecond
)

var (
	t0 = time.Now()
	t1 = t0.Add(time.Duration(1) * d)
	t2 = t0.Add(time.Duration(2) * d)
)

type allow struct {
	t  time.Time
	n  int
	ok bool
}

func run(t *testing.T, lim *Limiter, allows []allow) {
	t.Helper()
	for i, allow := range allows {
		ok := lim.AllowN(allow.t, allow.n)
		if ok != allow.ok {
			t.Errorf("step %d: lim.AllowN(%v, %v) = %v want %v %+v", i, allow.t, allow.n, ok, allow.ok, lim)
		}
	}
}

func TestLimiterBurst1(t *testing.T) {
	run(t, NewLimiter(10, 1), []allow{
		{t0, 1, true},
		{t0, 1, false},
		{t0, 1, false},
		{t1, 1, true},
		{t1, 1, false},
		{t1, 1, false},
		{t2, 2, false}, // burst size is 1, so n=2 always fails
		{t2, 1, true},
		{t2, 1, false},
	})
}

type wait struct {
	name   string
	n      int
	delay  int // in multiples of d
	nilErr bool
}

func runWait(t *testing.T, now time.Time, lim *Limiter, w wait) {
	t.Helper()
	start := now
	err := lim.WaitN(start, w.n, time.Second*10)
	delay := time.Since(start)

	if w.nilErr != err || !waitDelayOk(w.delay, delay) {
		errString := "<nil>"
		if !w.nilErr {
			errString = "<non-nil error>"
		}
		t.Errorf("lim.WaitN(%v, lim, %v) = %v with delay %v; want %v with delay %v (±%v)",
			w.name, w.n, err, delay, errString, d*time.Duration(w.delay), d/2)
	}
}

// dFromDuration converts a duration to the nearest multiple of the global constant d.
func dFromDuration(dur time.Duration) int {
	// Add d/2 to dur so that integer division will round to
	// the nearest multiple instead of truncating.
	// (We don't care about small inaccuracies.)
	return int((dur + (d / 2)) / d)
}

// waitDelayOk reports whether a duration spent in WaitN is “close enough” to
// wantD multiples of d, given scheduling slop.
func waitDelayOk(wantD int, got time.Duration) bool {
	gotD := dFromDuration(got)

	// The actual time spent waiting will be REDUCED by the amount of time spent
	// since the last call to the limiter. We expect the time in between calls to
	// be executing simple, straight-line, non-blocking code, so it should reduce
	// the wait time by no more than half a d, which would round to exactly wantD.
	if gotD < wantD {
		return false
	}

	// The actual time spend waiting will be INCREASED by the amount of scheduling
	// slop in the platform's sleep syscall, plus the amount of time spent executing
	// straight-line code before measuring the elapsed duration.
	//
	// The latter is surely less than half a d, but the former is empirically
	// sometimes larger on a number of platforms for a number of reasons.
	// NetBSD and OpenBSD tend to overshoot sleeps by a wide margin due to a
	// suspected platform bug; see https://go.dev/issue/44067 and
	// https://go.dev/issue/50189.
	// Longer delays were also also observed on slower builders with Linux kernels
	// (linux-ppc64le-buildlet, android-amd64-emu), and on Solaris and Plan 9.
	//
	// Since d is already fairly generous, we take 150% of wantD rounded up —
	// that's at least enough to account for the overruns we've seen so far in
	// practice.
	maxD := (wantD*3 + 1) / 2
	return gotD <= maxD
}

func TestWaitSimple(t *testing.T) {
	tt := time.Now()

	lim := NewLimiter(10, 3)

	runWait(t, tt, lim, wait{"act-now", 2, 0, true})
	runWait(t, tt, lim, wait{"act-later", 3, 2, true})
}
