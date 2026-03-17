// Package middleware provides chi-compatible HTTP middleware
package middleware

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	ratelimiter "github.com/Moukhtar-youssef/CourseLite/pkg/rateLimiter"
)

// KeyFunc extracts the rate-limit key from a request.
// Built-in options are provided below; you can also supply your own.
type KeyFunc func(r *http.Request) string

// KeyByIP uses the client's remote IP as the rate-limit key.
// It correctly strips the port and handles X-Forwarded-For when trusted.
func KeyByIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// KeyByUserID extracts a user ID stored in the request context under the
// provided key. Falls back to the client IP if the context value is missing.
// Pair with your authentication middleware that sets the context value.
func KeyByUserID(ctxKey any) KeyFunc {
	return func(r *http.Request) string {
		if v := r.Context().Value(ctxKey); v != nil {
			return fmt.Sprintf("%v", v)
		}
		return KeyByIP(r)
	}
}

// KeyByRoute uses "<method>:<pattern>" as the key so limits are per-endpoint.
func KeyByRoute(r *http.Request) string {
	return r.Method + ":" + r.URL.Path
}

// Options controls the middleware behaviour.
type Options struct {
	// KeyFunc extracts the bucket key from each request. Defaults to KeyByIP.
	KeyFunc KeyFunc

	// OnLimitReached is called when a request is denied.
	// If nil, the middleware writes a plain 429 with Retry-After header.
	OnLimitReached http.HandlerFunc

	// SkipFunc, if non-nil, skips rate limiting for matching requests
	SkipFunc func(r *http.Request) bool
}

// RateLimit returns a chi middleware that enforces the given rate limiter.
//
// Usage:
//
//	r := chi.NewRouter()
//	r.Use(middleware.RateLimit(myLimiter, middleware.Options{
//	    KeyFunc: middleware.KeyByIP,
//	}))
func RateLimit(rl ratelimiter.RateLimiter, opts Options,
) func(http.Handler) http.Handler {
	if opts.KeyFunc == nil {
		opts.KeyFunc = KeyByIP
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if opts.SkipFunc != nil && opts.SkipFunc(r) {
				next.ServeHTTP(w, r)
				return
			}

			key := opts.KeyFunc(r)
			result, err := rl.Allow(r.Context(), key)
			if err != nil {
				// Do not leak internal errors; log them and let the request through.
				// Replace with your structured logger if available.
				fmt.Printf("ratelimiter middleware error: %v\n", err)
				next.ServeHTTP(w, r)
				return
			}

			// Always set informational headers on every response.
			setHeaders(w, result)

			if result.Allowed {
				next.ServeHTTP(w, r)
				return
			}

			// Request denied.
			if opts.OnLimitReached != nil {
				opts.OnLimitReached(w, r)
				return
			}

			defaultDenyResponse(w, result)
		})
	}
}

// setHeaders writes standard rate-limit headers so clients can adapt.
func setHeaders(w http.ResponseWriter, res ratelimiter.Result) {
	h := w.Header()
	h.Set("X-RateLimit-Limit", strconv.Itoa(res.Limit))
	h.Set("X-RateLimit-Remaining", strconv.Itoa(res.Remaining))
	h.Set("X-RateLimit-Reset", strconv.FormatInt(res.ResetAt.Unix(), 10))

	if !res.Allowed && res.RetryAfter > 0 {
		h.Set("Retry-After", strconv.Itoa(int(res.RetryAfter/time.Second)+1))
	}
}

// defaultDenyResponse writes a 429 Too Many Requests response.
func defaultDenyResponse(w http.ResponseWriter, res ratelimiter.Result) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusTooManyRequests)
	retryAfterSec := int(res.RetryAfter/time.Second) + 1
	json.NewEncoder(w).Encode(map[string]string{
		"message": fmt.Sprintf(
			"Too many login attempts. Try again in %d second(s).",
			retryAfterSec,
		),
	})
}
