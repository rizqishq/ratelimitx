package ratelimitx

import (
	"net/http/httptest"
	"testing"
)

func TestByIP(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "203.0.113.10:8080"

	if got := ByIP()(req); got != "203.0.113.10" {
		t.Fatalf("unexpected ip key: %q", got)
	}
}

func TestByIPMalformedRemoteAddr(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "203.0.113.10"

	if got := ByIP()(req); got != "203.0.113.10" {
		t.Fatalf("unexpected ip key fallback: %q", got)
	}
}

func TestByHeader(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-API-Key", "abc123")

	if got := ByHeader("X-API-Key")(req); got != "abc123" {
		t.Fatalf("unexpected header key: %q", got)
	}
}
