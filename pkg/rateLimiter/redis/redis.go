// Package redis provides a Redis-backed, sliding-window rate limiter.
// It uses a Lua script executed atomically by Redis so the check-and-record
// operation is race-free even across multiple application instances.
//
// Requires: github.com/redis/go-redis/v9
package redisrl

import (
	"context"
	"fmt"
	"time"

	ratelimiter "github.com/Moukhtar-youssef/CourseLite/pkg/rateLimiter"
	"github.com/redis/go-redis/v9"
)

// slidingWindowScript is a Lua script that atomically:
//  1. Removes expired timestamps from the sorted set.
//  2. Counts the remaining (valid) members.
//  3. If under the limit, adds the current timestamp and returns 1 (allowed).
//  4. Otherwise returns 0 (denied) along with the oldest timestamp so the
//     caller can compute RetryAfter.
//
// KEYS[1]  – the rate-limit key in Redis
// ARGV[1]  – current Unix time in milliseconds
// ARGV[2]  – window duration in milliseconds
// ARGV[3]  – request limit (max requests per window)
// ARGV[4]  – TTL for the key in milliseconds
//
// Returns: {allowed (0|1), count, oldest_timestamp_ms}
const slidingWindowScript = `
local key        = KEYS[1]
local now        = tonumber(ARGV[1])
local window     = tonumber(ARGV[2])
local limit      = tonumber(ARGV[3])
local ttl        = tonumber(ARGV[4])
local window_start = now - window

-- Remove timestamps that have fallen outside the window.
redis.call('ZREMRANGEBYSCORE', key, '-inf', window_start)

local count = redis.call('ZCARD', key)
local oldest = 0

if count < limit then
    -- Record this request. Use now as score and a unique member.
    redis.call('ZADD', key, now, now .. '-' .. math.random(1, 1000000))
    redis.call('PEXPIRE', key, ttl)
    return {1, count + 1, oldest}
else
    local members = redis.call('ZRANGE', key, 0, 0, 'WITHSCORES')
    if #members >= 2 then
        oldest = tonumber(members[2])
    end
    return {0, count, oldest}
end
`

// RedisRateLimiter is a distributed, sliding-window rate limiter backed by
// Redis.
type RedisRateLimiter struct {
	cfg    ratelimiter.Config
	client redis.UniversalClient
	script *redis.Script
}

// New creates a RedisRateLimiter using the provided client.
// client can be a *redis.Client, *redis.ClusterClient, or *redis.Ring —
// anything that implements redis.UniversalClient.
func New(cfg ratelimiter.Config, client redis.UniversalClient,
) *RedisRateLimiter {
	return &RedisRateLimiter{
		cfg:    cfg,
		client: client,
		script: redis.NewScript(slidingWindowScript),
	}
}

// Allow implements ratelimiter.RateLimiter.
// It executes the Lua script atomically in Redis and translates the result
// into a Result value.
func (r *RedisRateLimiter) Allow(ctx context.Context, key string,
) (ratelimiter.Result, error) {
	fullKey := r.cfg.KeyPrefix + key
	now := time.Now()
	nowMs := now.UnixMilli()
	windowMs := r.cfg.Window.Milliseconds()
	// Keep the key alive for one full window beyond the last request.
	ttlMs := windowMs * 2

	vals, err := r.script.Run(
		ctx, r.client,
		[]string{fullKey},
		nowMs, windowMs, r.cfg.Limit, ttlMs,
	).Int64Slice()
	if err != nil {
		return ratelimiter.Result{}, fmt.Errorf(
			"ratelimiter/redis: lua script error: %w", err)
	}

	allowed := vals[0] == 1
	count := int(vals[1])
	oldestMs := vals[2]

	remaining := r.cfg.Limit - count
	if remaining < 0 {
		remaining = 0
	}

	resetAt := now.Add(r.cfg.Window)
	var retryAfter time.Duration

	if !allowed && oldestMs > 0 {
		oldest := time.UnixMilli(oldestMs)
		resetAt = oldest.Add(r.cfg.Window)
		retryAfter = time.Until(resetAt)
		if retryAfter < 0 {
			retryAfter = 0
		}
	}

	return ratelimiter.Result{
		Allowed:    allowed,
		Limit:      r.cfg.Limit,
		Remaining:  remaining,
		RetryAfter: retryAfter,
		ResetAt:    resetAt,
	}, nil
}

// Reset deletes the Redis key for the given key, immediately restoring
// the full quota.
func (r *RedisRateLimiter) Reset(ctx context.Context, key string) error {
	fullKey := r.cfg.KeyPrefix + key
	if err := r.client.Del(ctx, fullKey).Err(); err != nil {
		return fmt.Errorf("ratelimiter/redis: reset error: %w", err)
	}
	return nil
}

// Close is a no-op for the Redis implementation because the caller owns the
// client lifecycle. Close the redis.Client itself when shutting down.
func (r *RedisRateLimiter) Close() error { return nil }
