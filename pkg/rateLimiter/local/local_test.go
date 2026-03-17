package local_test

import (
	"context"
	"testing"
	"time"

	ratelimiter "github.com/Moukhtar-youssef/CourseLite/pkg/rateLimiter"
	"github.com/Moukhtar-youssef/CourseLite/pkg/rateLimiter/local"
)

func TestAllow_UnderLimit(t *testing.T) {
	rl := local.New(ratelimiter.Config{Limit: 3, Window: time.Second})
	defer rl.Close()

	ctx := context.Background()
	for i := range 3 {
		res, err := rl.Allow(ctx, "user1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !res.Allowed {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}
}

func TestAllow_ExceedsLimit(t *testing.T) {
	rl := local.New(ratelimiter.Config{Limit: 2, Window: time.Second})
	defer rl.Close()

	ctx := context.Background()
	for range 2 {
		rl.Allow(ctx, "user2") //nolint:errcheck
	}

	res, err := rl.Allow(ctx, "user2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Allowed {
		t.Fatal("third request should be denied")
	}
	if res.Remaining != 0 {
		t.Fatalf("expected 0 remaining, got %d", res.Remaining)
	}
	if res.RetryAfter <= 0 {
		t.Fatal("expected a positive RetryAfter duration")
	}
}

func TestReset(t *testing.T) {
	rl := local.New(ratelimiter.Config{Limit: 1, Window: time.Second})
	defer rl.Close()

	ctx := context.Background()
	rl.Allow(ctx, "user3") //nolint:errcheck

	res, _ := rl.Allow(ctx, "user3")
	if res.Allowed {
		t.Fatal("second request should be denied before reset")
	}

	if err := rl.Reset(ctx, "user3"); err != nil {
		t.Fatalf("reset failed: %v", err)
	}

	res, _ = rl.Allow(ctx, "user3")
	if !res.Allowed {
		t.Fatal("request after reset should be allowed")
	}
}

func TestKeyPrefix(t *testing.T) {
	rl := local.New(ratelimiter.Config{
		Limit: 1, Window: time.Second,
		KeyPrefix: "test:",
	})
	defer rl.Close()

	ctx := context.Background()
	res1, _ := rl.Allow(ctx, "x")
	if !res1.Allowed {
		t.Fatal("first request should be allowed")
	}
	res2, _ := rl.Allow(ctx, "x")
	if res2.Allowed {
		t.Fatal("second request should be denied")
	}
}

func TestConcurrentAccess(t *testing.T) {
	rl := local.New(ratelimiter.Config{Limit: 100, Window: time.Second})
	defer rl.Close()

	ctx := context.Background()
	done := make(chan struct{})

	for range 50 {
		go func() {
			rl.Allow(ctx, "concurrent-key") //nolint:errcheck
			done <- struct{}{}
		}()
	}
	for range 50 {
		<-done
	}
}
