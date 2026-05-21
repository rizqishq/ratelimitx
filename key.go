package ratelimitx

import (
	"net"
	"net/http"
)

// KeyFunc extracts a limiter key from an HTTP request.
type KeyFunc func(r *http.Request) string

// ByIP extracts the request key from the remote IP address.
func ByIP() KeyFunc {
	return func(r *http.Request) string {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			return r.RemoteAddr
		}

		return host
	}
}

// ByHeader extracts the request key from a specific header.
func ByHeader(headerName string) KeyFunc {
	return func(r *http.Request) string {
		return r.Header.Get(headerName)
	}
}
