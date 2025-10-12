package ratelimit

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// RateLimiter implements a token bucket rate limiter
type RateLimiter struct {
	mu       sync.Mutex
	tokens   int
	capacity int
	refillRate int // tokens per second
	lastRefill time.Time
	enabled    bool
}

// New creates a new rate limiter
func New(rps int, enabled bool) *RateLimiter {
	return &RateLimiter{
		tokens:     rps,
		capacity:   rps,
		refillRate: rps,
		lastRefill: time.Now(),
		enabled:    enabled,
	}
}

// Allow checks if a request is allowed under the current rate limit
func (rl *RateLimiter) Allow() bool {
	if !rl.enabled {
		return true
	}
	
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	now := time.Now()
	elapsed := now.Sub(rl.lastRefill)
	
	// Refill tokens based on elapsed time
	tokensToAdd := int(elapsed.Seconds() * float64(rl.refillRate))
	if tokensToAdd > 0 {
		rl.tokens += tokensToAdd
		if rl.tokens > rl.capacity {
			rl.tokens = rl.capacity
		}
		rl.lastRefill = now
	}
	
	// Check if we have tokens available
	if rl.tokens > 0 {
		rl.tokens--
		return true
	}
	
	return false
}

// Wait blocks until a token is available or context is cancelled
func (rl *RateLimiter) Wait(ctx context.Context) error {
	if !rl.enabled {
		return nil
	}
	
	for {
		if rl.Allow() {
			return nil
		}
		
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Millisecond * 10): // Check every 10ms
			continue
		}
	}
}

// Middleware wraps a function with rate limiting
func (rl *RateLimiter) Middleware(fn func(ctx context.Context) error) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		if !rl.Allow() {
			return fmt.Errorf("rate limit exceeded")
		}
		return fn(ctx)
	}
}

// GetStats returns current rate limiter statistics
func (rl *RateLimiter) GetStats() RateLimiterStats {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	return RateLimiterStats{
		Enabled:       rl.enabled,
		CurrentTokens: rl.tokens,
		Capacity:      rl.capacity,
		RefillRate:    rl.refillRate,
	}
}

// RateLimiterStats holds statistics about the rate limiter
type RateLimiterStats struct {
	Enabled       bool
	CurrentTokens int
	Capacity      int
	RefillRate    int
}
