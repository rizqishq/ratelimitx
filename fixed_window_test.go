package ratelimitx

import (
	"testing"
	"time"
)

func TestNewFixedWindowLimiterValidation(t *testing.T) {
	t.Parallel()

	_, err := NewFixedWindowLimiter(0, time.Second)
	if err == nil {
		t.Fatal("expected error for zero limit")
	}

	_, err = NewFixedWindowLimiter(1, 0)
	if err == nil {
		t.Fatal("expected error for zero window")
	}
}

func TestFixedWindowLimiterAllowAndReset(t *testing.T) {
	limiter, err := NewFixedWindowLimiter(2, 50*time.Millisecond)
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
	if third.Allowed || third.Remaining != 0 || third.RetryAfter <= 0 {
		t.Fatalf("unexpected third result: %+v", third)
	}

	time.Sleep(60 * time.Millisecond)

	reset := limiter.Allow("client-a")
	if !reset.Allowed || reset.Remaining != 1 {
		t.Fatalf("unexpected reset result: %+v", reset)
	}
}

func TestFixedWindowLimiterTracksKeysSeparately(t *testing.T) {
	limiter, err := NewFixedWindowLimiter(1, time.Second)
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

func TestFixedWindowLimiterSweepsExpiredEntries(t *testing.T) {
	limiter, err := NewFixedWindowLimiter(1, time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	now := time.Now()
	limiter.clients["expired"] = &fixedWindowEntry{count: 1, expiresAt: now.Add(-time.Second)}
	limiter.clients["active"] = &fixedWindowEntry{count: 1, expiresAt: now.Add(time.Second)}
	limiter.nextSweep = now.Add(-time.Millisecond)

	limiter.Allow("fresh")

	if _, ok := limiter.clients["expired"]; ok {
		t.Fatal("expected expired key to be removed")
	}

	if _, ok := limiter.clients["active"]; !ok {
		t.Fatal("expected active key to remain")
	}
}
