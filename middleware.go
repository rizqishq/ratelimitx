package ratelimitx

import (
	"encoding/json"
	"math"
	"net/http"
	"strconv"
)

// HTTPMiddleware wraps an http.Handler with rate limiting behavior.
func HTTPMiddleware(limiter Limiter, keyFunc KeyFunc) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := keyFunc(r)

			result := limiter.Allow(key)

			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(result.Limit))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(result.ResetAt.Unix(), 10))

			if !result.Allowed {
				retryAfter := max(int(math.Ceil(result.RetryAfter.Seconds())), 1)

				w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)

				_ = json.NewEncoder(w).Encode(map[string]string{
					"error": "rate limit exceeded",
				})

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
