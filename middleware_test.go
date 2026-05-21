package ratelimitx

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type stubLimiter struct {
	result Result
	key    string
}

func (s *stubLimiter) Allow(key string) Result {
	s.key = key
	return s.result
}

func TestHTTPMiddlewareAllowsRequest(t *testing.T) {
	limiter := &stubLimiter{result: Result{
		Allowed:   true,
		Limit:     5,
		Remaining: 4,
		ResetAt:   time.Unix(1700000000, 0),
	}}

	middleware := HTTPMiddleware(limiter, ByHeader("X-API-Key"))
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-API-Key", "client-1")
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if limiter.key != "client-1" {
		t.Fatalf("unexpected limiter key: %q", limiter.key)
	}

	if res.Code != http.StatusNoContent {
		t.Fatalf("unexpected status code: %d", res.Code)
	}

	if got := res.Header().Get("X-RateLimit-Limit"); got != "5" {
		t.Fatalf("unexpected limit header: %q", got)
	}

	if got := res.Header().Get("X-RateLimit-Remaining"); got != "4" {
		t.Fatalf("unexpected remaining header: %q", got)
	}

	if got := res.Header().Get("X-RateLimit-Reset"); got != "1700000000" {
		t.Fatalf("unexpected reset header: %q", got)
	}
}

func TestHTTPMiddlewareBlocksRequest(t *testing.T) {
	limiter := &stubLimiter{result: Result{
		Allowed:    false,
		Limit:      5,
		Remaining:  0,
		RetryAfter: 1500 * time.Millisecond,
		ResetAt:    time.Unix(1700000000, 0),
	}}

	middleware := HTTPMiddleware(limiter, ByHeader("X-API-Key"))
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called")
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-API-Key", "client-1")
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if res.Code != http.StatusTooManyRequests {
		t.Fatalf("unexpected status code: %d", res.Code)
	}

	if got := res.Header().Get("Retry-After"); got != "2" {
		t.Fatalf("unexpected retry-after header: %q", got)
	}

	if got := res.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("unexpected content-type: %q", got)
	}

	var body map[string]string
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("unexpected json body: %v", err)
	}

	if body["error"] != "rate limit exceeded" {
		t.Fatalf("unexpected error body: %+v", body)
	}
}
