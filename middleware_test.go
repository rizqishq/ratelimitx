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

func TestWrapHTTPAllowsRequest(t *testing.T) {
	limiter := &stubLimiter{result: Result{
		Allowed:   true,
		Limit:     5,
		Remaining: 4,
		ResetAt:   time.Unix(1700000000, 0),
	}}

	handler := WrapHTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}), limiter, ByHeader("X-API-Key"))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-API-Key", "client-wrap")
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if limiter.key != "client-wrap" {
		t.Fatalf("unexpected limiter key: %q", limiter.key)
	}

	if res.Code != http.StatusNoContent {
		t.Fatalf("unexpected status code: %d", res.Code)
	}

	if got := res.Header().Get("X-RateLimit-Limit"); got != "5" {
		t.Fatalf("unexpected limit header: %q", got)
	}
}

func TestWrapHTTPBlocksRequest(t *testing.T) {
	limiter := &stubLimiter{result: Result{
		Allowed:    false,
		Limit:      5,
		Remaining:  0,
		RetryAfter: 1500 * time.Millisecond,
		ResetAt:    time.Unix(1700000000, 0),
	}}

	handler := WrapHTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called")
	}), limiter, ByHeader("X-API-Key"))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-API-Key", "client-wrap")
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if res.Code != http.StatusTooManyRequests {
		t.Fatalf("unexpected status code: %d", res.Code)
	}

	if got := res.Header().Get("Retry-After"); got != "2" {
		t.Fatalf("unexpected retry-after header: %q", got)
	}
}

func TestWrapHTTPWithCustomRejectedResponse(t *testing.T) {
	limiter := &stubLimiter{result: Result{
		Allowed:    false,
		Limit:      3,
		Remaining:  0,
		RetryAfter: 3 * time.Second,
		ResetAt:    time.Unix(1700001234, 0),
	}}

	var called bool
	var receivedResult Result

	handler := WrapHTTPWith(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called")
	}), limiter, ByHeader("X-API-Key"), HTTPOptions{
		OnRejected: func(w http.ResponseWriter, r *http.Request, result Result) {
			called = true
			receivedResult = result
			w.Header().Set("Content-Type", "text/plain")
			w.Header().Set("X-Custom-Rejected", "yes")
			w.WriteHeader(http.StatusTeapot)
			_, _ = w.Write([]byte("blocked by wrap helper"))
		},
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-API-Key", "client-wrap-custom")
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if !called {
		t.Fatal("expected custom rejection handler to be called")
	}

	if receivedResult != limiter.result {
		t.Fatalf("unexpected result passed to custom handler: %+v", receivedResult)
	}

	if res.Code != http.StatusTeapot {
		t.Fatalf("unexpected status code: %d", res.Code)
	}

	if got := res.Header().Get("X-Custom-Rejected"); got != "yes" {
		t.Fatalf("unexpected custom header: %q", got)
	}

	if got := res.Header().Get("X-RateLimit-Limit"); got != "3" {
		t.Fatalf("unexpected limit header: %q", got)
	}

	if body := res.Body.String(); body != "blocked by wrap helper" {
		t.Fatalf("unexpected body: %q", body)
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

func TestHTTPMiddlewareWithCustomRejectedResponse(t *testing.T) {
	limiter := &stubLimiter{result: Result{
		Allowed:    false,
		Limit:      3,
		Remaining:  0,
		RetryAfter: 3 * time.Second,
		ResetAt:    time.Unix(1700001234, 0),
	}}

	var called bool
	var receivedResult Result
	var receivedPath string

	middleware := HTTPMiddlewareWith(limiter, ByHeader("X-API-Key"), HTTPOptions{
		OnRejected: func(w http.ResponseWriter, r *http.Request, result Result) {
			called = true
			receivedResult = result
			receivedPath = r.URL.Path
			w.Header().Set("Content-Type", "text/plain")
			w.Header().Set("X-Custom-Rejected", "yes")
			w.WriteHeader(http.StatusTeapot)
			_, _ = w.Write([]byte("blocked by custom handler"))
		},
	})

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called")
	}))

	req := httptest.NewRequest("GET", "/limited", nil)
	req.Header.Set("X-API-Key", "client-2")
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if !called {
		t.Fatal("expected custom rejection handler to be called")
	}

	if receivedPath != "/limited" {
		t.Fatalf("unexpected request path: %q", receivedPath)
	}

	if receivedResult != limiter.result {
		t.Fatalf("unexpected result passed to custom handler: %+v", receivedResult)
	}

	if res.Code != http.StatusTeapot {
		t.Fatalf("unexpected status code: %d", res.Code)
	}

	if got := res.Header().Get("X-Custom-Rejected"); got != "yes" {
		t.Fatalf("unexpected custom header: %q", got)
	}

	if got := res.Header().Get("X-RateLimit-Limit"); got != "3" {
		t.Fatalf("unexpected limit header: %q", got)
	}

	if got := res.Header().Get("X-RateLimit-Remaining"); got != "0" {
		t.Fatalf("unexpected remaining header: %q", got)
	}

	if got := res.Header().Get("X-RateLimit-Reset"); got != "1700001234" {
		t.Fatalf("unexpected reset header: %q", got)
	}

	if body := res.Body.String(); body != "blocked by custom handler" {
		t.Fatalf("unexpected body: %q", body)
	}
}

func TestHTTPMiddlewareWithUsesDefaultRejectedResponseWhenNil(t *testing.T) {
	limiter := &stubLimiter{result: Result{
		Allowed:    false,
		Limit:      2,
		Remaining:  0,
		RetryAfter: time.Second,
		ResetAt:    time.Unix(1700005678, 0),
	}}

	middleware := HTTPMiddlewareWith(limiter, ByHeader("X-API-Key"), HTTPOptions{})
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called")
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-API-Key", "client-3")
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if res.Code != http.StatusTooManyRequests {
		t.Fatalf("unexpected status code: %d", res.Code)
	}

	if got := res.Header().Get("Retry-After"); got != "1" {
		t.Fatalf("unexpected retry-after header: %q", got)
	}
}
