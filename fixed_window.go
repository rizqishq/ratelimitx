package ratelimitx

import (
	"errors"
	"sync"
	"time"
)

// FixedWindowLimiter applies a fixed-window rate limit in local memory.
type FixedWindowLimiter struct {
	limit  int
	window time.Duration

	mu        sync.Mutex
	clients   map[string]*fixedWindowEntry
	nextSweep time.Time
}

type fixedWindowEntry struct {
	count     int
	expiresAt time.Time
}

// NewFixedWindowLimiter creates an in-memory fixed-window limiter.
func NewFixedWindowLimiter(limit int, window time.Duration) (*FixedWindowLimiter, error) {
	if limit <= 0 {
		return nil, errors.New("ratelimitx: limit must be greater than 0")
	}

	if window <= 0 {
		return nil, errors.New("ratelimitx: window must be greater than 0")
	}

	now := time.Now()

	return &FixedWindowLimiter{
		limit:     limit,
		window:    window,
		clients:   make(map[string]*fixedWindowEntry),
		nextSweep: now.Add(window),
	}, nil
}

// Allow reports whether the key is currently allowed.
func (l *FixedWindowLimiter) Allow(key string) Result {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	l.sweepExpired(now)

	entry, exists := l.clients[key]

	if !exists || !now.Before(entry.expiresAt) {
		resetAt := now.Add(l.window)

		l.clients[key] = &fixedWindowEntry{
			count:     1,
			expiresAt: resetAt,
		}

		return Result{
			Allowed:    true,
			Limit:      l.limit,
			Remaining:  l.limit - 1,
			RetryAfter: 0,
			ResetAt:    resetAt,
		}
	}

	if entry.count >= l.limit {
		return Result{
			Allowed:    false,
			Limit:      l.limit,
			Remaining:  0,
			RetryAfter: time.Until(entry.expiresAt),
			ResetAt:    entry.expiresAt,
		}
	}

	entry.count++

	return Result{
		Allowed:    true,
		Limit:      l.limit,
		Remaining:  l.limit - entry.count,
		RetryAfter: 0,
		ResetAt:    entry.expiresAt,
	}
}

func (l *FixedWindowLimiter) sweepExpired(now time.Time) {
	if now.Before(l.nextSweep) {
		return
	}

	for key, entry := range l.clients {
		if !now.Before(entry.expiresAt) {
			delete(l.clients, key)
		}
	}

	l.nextSweep = now.Add(l.window)
}
