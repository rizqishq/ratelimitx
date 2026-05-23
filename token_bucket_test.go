package ratelimitx

import (
	"testing"
	"time"
)

func TestNewTokenBucketValidation(t *testing.T) {
	t.Parallel()

	_, err := NewTokenBucket(0, time.Second)
	if err == nil {
		t.Fatal("expected error for zero capacity")
	}

	_, err = NewTokenBucket(1, 0)
	if err == nil {
		t.Fatal("expected error for zero refill time")
	}
}

func TestTokenBucketAllowsBurstAndBlocks(t *testing.T) {
	limiter, err := NewTokenBucket(2, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	first := limiter.Allow("client-a")
	if !first.Allowed || first.Remaining != 1 {
		t.Fatalf("unexpected first result: %+v", first)
	}

	second := limiter.Allow("client-a")
	if !second.Allowed || second.Remaining != 0 {
		t.Fatalf("unexpected second result: %+v", second)
	}

	third := limiter.Allow("client-a")
	if third.Allowed || third.RetryAfter <= 0 {
		t.Fatalf("unexpected third result: %+v", third)
	}
}

func TestTokenBucketRefillsAfterTime(t *testing.T) {
	limiter, err := NewTokenBucket(1, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result := limiter.Allow("client-a"); !result.Allowed {
		t.Fatalf("expected first request to pass: %+v", result)
	}

	if result := limiter.Allow("client-a"); result.Allowed {
		t.Fatalf("expected second request to block: %+v", result)
	}

	time.Sleep(60 * time.Millisecond)

	if result := limiter.Allow("client-a"); !result.Allowed {
		t.Fatalf("expected request after refill to pass: %+v", result)
	}
}

func TestTokenBucketTracksKeysSeparately(t *testing.T) {
	limiter, err := NewTokenBucket(1, time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result := limiter.Allow("client-a"); !result.Allowed {
		t.Fatalf("client-a should be allowed: %+v", result)
	}

	if result := limiter.Allow("client-b"); !result.Allowed {
		t.Fatalf("client-b should be allowed: %+v", result)
	}

	if result := limiter.Allow("client-a"); result.Allowed {
		t.Fatalf("client-a should be blocked on second request: %+v", result)
	}
}

func TestTokenBucketSweepsFullIdleEntries(t *testing.T) {
	limiter, err := NewTokenBucket(2, time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	now := time.Now()
	limiter.entries["idle"] = &tokenBucketEntry{tokens: 0, lastRefill: now.Add(-2 * time.Second)}
	limiter.entries["active"] = &tokenBucketEntry{tokens: 0, lastRefill: now}
	limiter.nextSweep = now.Add(-time.Millisecond)

	limiter.Allow("fresh")

	if _, ok := limiter.entries["idle"]; ok {
		t.Fatal("expected full idle key to be removed")
	}

	if _, ok := limiter.entries["active"]; !ok {
		t.Fatal("expected active key to remain")
	}
}
