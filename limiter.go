package ratelimitx

import "time"

// Limiter decides whether a key may proceed at the current time.
type Limiter interface {
	Allow(key string) Result
}

// Result describes the outcome of a limiter decision.
type Result struct {
	Allowed    bool
	Limit      int
	Remaining  int
	RetryAfter time.Duration
	ResetAt    time.Time
}
