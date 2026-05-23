package ratelimitx

import (
	"errors"
	"math"
	"sync"
	"time"
)

// TokenBucketLimiter applies a token bucket rate limit in local memory.
type TokenBucketLimiter struct {
	capacity   int
	refillTime time.Duration
	rate       float64

	mu        sync.Mutex
	entries   map[string]*tokenBucketEntry
	nextSweep time.Time
}

type tokenBucketEntry struct {
	tokens     float64
	lastRefill time.Time
}

// NewTokenBucket creates an in-memory token bucket limiter.
func NewTokenBucket(capacity int, refillTime time.Duration) (*TokenBucketLimiter, error) {
	if capacity <= 0 {
		return nil, errors.New("ratelimitx: capacity must be greater than 0")
	}

	if refillTime <= 0 {
		return nil, errors.New("ratelimitx: refill time must be greater than 0")
	}

	now := time.Now()

	return &TokenBucketLimiter{
		capacity:   capacity,
		refillTime: refillTime,
		rate:       float64(capacity) / refillTime.Seconds(),
		entries:    make(map[string]*tokenBucketEntry),
		nextSweep:  now.Add(refillTime),
	}, nil
}

// Allow reports whether the key is currently allowed.
func (l *TokenBucketLimiter) Allow(key string) Result {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	l.sweepIdle(now)

	entry, ok := l.entries[key]
	if !ok {
		entry = &tokenBucketEntry{
			tokens:     float64(l.capacity),
			lastRefill: now,
		}
		l.entries[key] = entry
	}

	l.refill(entry, now)

	if entry.tokens >= 1 {
		entry.tokens--

		return Result{
			Allowed:    true,
			Limit:      l.capacity,
			Remaining:  int(math.Floor(entry.tokens)),
			RetryAfter: 0,
			ResetAt:    l.fullAt(now, entry.tokens),
		}
	}

	return Result{
		Allowed:    false,
		Limit:      l.capacity,
		Remaining:  0,
		RetryAfter: l.retryAfter(entry.tokens),
		ResetAt:    l.fullAt(now, entry.tokens),
	}
}

func (l *TokenBucketLimiter) refill(entry *tokenBucketEntry, now time.Time) {
	elapsed := now.Sub(entry.lastRefill)
	if elapsed <= 0 {
		return
	}

	entry.tokens += elapsed.Seconds() * l.rate
	if entry.tokens > float64(l.capacity) {
		entry.tokens = float64(l.capacity)
	}
	entry.lastRefill = now
}

func (l *TokenBucketLimiter) retryAfter(tokens float64) time.Duration {
	missingTokens := 1 - tokens
	if missingTokens <= 0 {
		return 0
	}

	nanoseconds := math.Ceil((missingTokens / l.rate) * float64(time.Second))
	return time.Duration(nanoseconds)
}

func (l *TokenBucketLimiter) fullAt(now time.Time, tokens float64) time.Time {
	missingTokens := float64(l.capacity) - tokens
	if missingTokens <= 0 {
		return now
	}

	nanoseconds := math.Ceil((missingTokens / l.rate) * float64(time.Second))
	return now.Add(time.Duration(nanoseconds))
}

func (l *TokenBucketLimiter) sweepIdle(now time.Time) {
	if now.Before(l.nextSweep) {
		return
	}

	for key, entry := range l.entries {
		shadow := *entry
		l.refill(&shadow, now)
		if shadow.tokens >= float64(l.capacity) {
			delete(l.entries, key)
		}
	}

	l.nextSweep = now.Add(l.refillTime)
}
