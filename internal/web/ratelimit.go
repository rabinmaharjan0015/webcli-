package web

import (
	"sync"
	"time"
)

type RateLimiter struct {
	mu        sync.Mutex
	rate      int
	burst     int
	tokens    int
	lastCheck time.Time
	interval  time.Duration
}

func NewRateLimiter(requestsPerMinute, burst int) *RateLimiter {
	if requestsPerMinute <= 0 {
		requestsPerMinute = 30
	}
	if burst <= 0 {
		burst = 5
	}
	return &RateLimiter{
		rate:      requestsPerMinute,
		burst:     burst,
		tokens:    burst,
		lastCheck: time.Now(),
		interval:  time.Minute / time.Duration(requestsPerMinute),
	}
}

func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(rl.lastCheck)
	rl.lastCheck = now

	rl.tokens += int(elapsed / rl.interval)
	if rl.tokens > rl.burst {
		rl.tokens = rl.burst
	}

	if rl.tokens > 0 {
		rl.tokens--
		return true
	}

	return false
}

func (rl *RateLimiter) Wait() {
	for !rl.Allow() {
		time.Sleep(rl.interval)
	}
}
