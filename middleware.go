package ratelimitx

import (
	"encoding/json"
	"math"
	"net/http"
	"strconv"
)

// RateLimitResponseFunc writes the HTTP response for a rejected request.
type RateLimitResponseFunc func(w http.ResponseWriter, r *http.Request, result Result)

// HTTPOptions configures optional HTTP middleware behavior.
type HTTPOptions struct {
	OnRejected RateLimitResponseFunc
}

// WrapHTTP wraps an http.Handler with rate limiting behavior.
func WrapHTTP(next http.Handler, limiter Limiter, keyFunc KeyFunc) http.Handler {
	return WrapHTTPWith(next, limiter, keyFunc, HTTPOptions{})
}

// WrapHTTPWith wraps an http.Handler with rate limiting behavior and optional middleware customization.
func WrapHTTPWith(next http.Handler, limiter Limiter, keyFunc KeyFunc, options HTTPOptions) http.Handler {
	return HTTPMiddlewareWith(limiter, keyFunc, options)(next)
}

// HTTPMiddleware wraps an http.Handler with rate limiting behavior.
func HTTPMiddleware(limiter Limiter, keyFunc KeyFunc) func(http.Handler) http.Handler {
	return HTTPMiddlewareWith(limiter, keyFunc, HTTPOptions{})
}

// HTTPMiddlewareWith wraps an http.Handler with rate limiting behavior and optional middleware customization.
func HTTPMiddlewareWith(limiter Limiter, keyFunc KeyFunc, options HTTPOptions) func(http.Handler) http.Handler {
	onRejected := options.OnRejected
	if onRejected == nil {
		onRejected = defaultRejectedResponse
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := keyFunc(r)

			result := limiter.Allow(key)

			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(result.Limit))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(result.ResetAt.Unix(), 10))

			if !result.Allowed {
				onRejected(w, r, result)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func defaultRejectedResponse(w http.ResponseWriter, _ *http.Request, result Result) {
	retryAfter := max(int(math.Ceil(result.RetryAfter.Seconds())), 1)

	w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusTooManyRequests)

	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": "rate limit exceeded",
	})
}
