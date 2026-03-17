// Package ratelimiter defines the core interface and types for rate limiting
// in the CourseLite application. Implementations can be swapped between
// local (in-memory) and Redis-backed strategies.
package ratelimiter

import (
	"context"
	"time"
)

// Result holds the outcome of a single rate limit check.
type Result struct {
	// Allowed indicates whether the request should be permitted.
	Allowed bool

	// Limit is the maximum number of requests allowed in the window.
	Limit int

	// Remaining is how many requests are left in the current window.
	Remaining int

	// RetryAfter is how long the caller should wait before retrying
	// when Allowed is false. Zero means no information available.
	RetryAfter time.Duration

	// ResetAt is the absolute time when the window resets.
	ResetAt time.Time
}

// Config holds common configuration for any rate limiter implementation.
type Config struct {
	// Limit is the maximum number of requests allowed per window.
	Limit int

	// Window is the duration of the sliding/fixed window.
	Window time.Duration

	// KeyPrefix is an optional namespace prefix applied to every key
	// (useful to separate rate limiters by feature, e.g. "api:", "login:").
	KeyPrefix string
}

// RateLimiter is the interface every backend must satisfy.
// Implementations must be safe for concurrent use.
type RateLimiter interface {
	// Allow checks and records a request for the given key.
	// key is typically a user ID, IP address, or composite identifier.
	Allow(ctx context.Context, key string) (Result, error)

	// Reset clears the rate limit state for the given key immediately.
	Reset(ctx context.Context, key string) error

	// Close releases any resources held by the implementation (connections, goroutines, etc.).
	Close() error
}
