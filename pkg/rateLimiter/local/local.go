// Package local provides an in-memory, sliding-window rate limiter.
// It is suitable for single-instance deployments or development/testing.
// For multi-instance production deployments use the redis implementation.
package local

import (
	"context"
	"sync"
	"time"

	ratelimiter "github.com/Moukhtar-youssef/CourseLite/pkg/rateLimiter"
)

// entry tracks the request timestamps for one key.
type entry struct {
	mu         sync.Mutex
	timestamps []time.Time
}

// LocalRateLimiter is a thread-safe, in-process sliding-window rate limiter.
type LocalRateLimiter struct {
	cfg    ratelimiter.Config
	mu     sync.RWMutex
	store  map[string]*entry
	stopCh chan struct{}
}

// New creates a LocalRateLimiter and starts a background goroutine that
// periodically evicts stale keys to prevent unbounded memory growth.
func New(cfg ratelimiter.Config) *LocalRateLimiter {
	l := &LocalRateLimiter{
		cfg:    cfg,
		store:  make(map[string]*entry),
		stopCh: make(chan struct{}),
	}
	go l.cleanup()
	return l
}

// Allow implements ratelimiter.RateLimiter using a sliding window algorithm.
// It records the current timestamp and counts how many requests fall within
// the configured window, then decides whether to allow or deny.
func (l *LocalRateLimiter) Allow(_ context.Context, key string,
) (ratelimiter.Result, error) {
	fullKey := l.cfg.KeyPrefix + key
	now := time.Now()
	windowStart := now.Add(-l.cfg.Window)

	e := l.getOrCreate(fullKey)
	e.mu.Lock()
	defer e.mu.Unlock()

	// Evict timestamps outside the window (sliding window).
	valid := e.timestamps[:0]
	for _, ts := range e.timestamps {
		if ts.After(windowStart) {
			valid = append(valid, ts)
		}
	}
	e.timestamps = valid

	remaining := l.cfg.Limit - len(e.timestamps)
	resetAt := now.Add(l.cfg.Window)

	if remaining <= 0 {
		// Earliest request in the window tells us when a slot opens up.
		var retryAfter time.Duration
		if len(e.timestamps) > 0 {
			retryAfter = e.timestamps[0].Add(l.cfg.Window).Sub(now)
			resetAt = e.timestamps[0].Add(l.cfg.Window)
		}
		return ratelimiter.Result{
			Allowed:    false,
			Limit:      l.cfg.Limit,
			Remaining:  0,
			RetryAfter: retryAfter,
			ResetAt:    resetAt,
		}, nil
	}

	// Permit and record this request.
	e.timestamps = append(e.timestamps, now)
	remaining--

	return ratelimiter.Result{
		Allowed:   true,
		Limit:     l.cfg.Limit,
		Remaining: remaining,
		ResetAt:   resetAt,
	}, nil
}

// Reset removes all recorded timestamps for key, immediately restoring
// the full quota. Useful for tests or admin operations.
func (l *LocalRateLimiter) Reset(_ context.Context, key string) error {
	fullKey := l.cfg.KeyPrefix + key

	l.mu.Lock()
	defer l.mu.Unlock()

	if e, ok := l.store[fullKey]; ok {
		e.mu.Lock()
		e.timestamps = e.timestamps[:0]
		e.mu.Unlock()
	}
	return nil
}

// Close stops the background cleanup goroutine and releases memory.
func (l *LocalRateLimiter) Close() error {
	close(l.stopCh)
	l.mu.Lock()
	l.store = nil
	l.mu.Unlock()
	return nil
}

// getOrCreate retrieves or lazily initialises the entry for fullKey.
func (l *LocalRateLimiter) getOrCreate(fullKey string) *entry {
	l.mu.RLock()
	e, ok := l.store[fullKey]
	l.mu.RUnlock()
	if ok {
		return e
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	// Double-checked locking.
	if e, ok = l.store[fullKey]; ok {
		return e
	}
	e = &entry{}
	l.store[fullKey] = e
	return e
}

// cleanup runs every window duration and removes keys whose last request
// is older than one window, keeping memory bounded.
func (l *LocalRateLimiter) cleanup() {
	ticker := time.NewTicker(l.cfg.Window)
	defer ticker.Stop()

	for {
		select {
		case <-l.stopCh:
			return
		case now := <-ticker.C:
			cutoff := now.Add(-l.cfg.Window)

			l.mu.Lock()
			for key, e := range l.store {
				e.mu.Lock()
				if len(e.timestamps) == 0 || e.timestamps[len(e.timestamps)-1].Before(cutoff) {
					delete(l.store, key)
				}
				e.mu.Unlock()
			}
			l.mu.Unlock()
		}
	}
}
