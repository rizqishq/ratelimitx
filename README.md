# ratelimitx

`ratelimitx` is a small Go rate limiter library for `net/http` applications.

It is built for practical personal use: small enough to understand quickly, simple enough to adapt, and focused enough to stay out of the way.

It currently provides:
- an in-memory fixed-window limiter
- request key helpers
- an HTTP middleware that returns `429 Too Many Requests`

## Install

```bash
go get github.com/rizqishq/ratelimitx
```

## Usage

```go
package main

import (
    "log"
    "net/http"
    "time"

    "github.com/rizqishq/ratelimitx"
)

func main() {
    limiter, err := ratelimitx.NewFixedWindowLimiter(5, time.Minute)
    if err != nil {
        log.Fatal(err)
    }

    mux := http.NewServeMux()
    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        _, _ = w.Write([]byte("ok\n"))
    })

    handler := ratelimitx.WrapHTTP(mux, limiter, ratelimitx.ByIP())

    if err := http.ListenAndServe(":8080", handler); err != nil {
        log.Fatal(err)
    }
}
```

## Key Functions

Use the built-in key helpers:

```go
ratelimitx.ByIP()
ratelimitx.ByHeader("X-API-Key")
```

You can also provide your own `KeyFunc`.

## Middleware Options

For the simplest `net/http` usage:

```go
handler := ratelimitx.WrapHTTP(mux, limiter, ratelimitx.ByIP())
```

If you want the middleware function itself:

```go
middleware := ratelimitx.HTTPMiddleware(limiter, ratelimitx.ByIP())
```

Or customize the rejected response while keeping the rate-limit headers:

```go
middleware := ratelimitx.HTTPMiddlewareWithOptions(
    limiter,
    ratelimitx.ByIP(),
    ratelimitx.HTTPMiddlewareOptions{
        OnRejected: func(w http.ResponseWriter, r *http.Request, result ratelimitx.Result) {
            w.Header().Set("Content-Type", "text/plain")
            w.WriteHeader(http.StatusTooManyRequests)
            _, _ = w.Write([]byte("chill bruh"))
        },
    },
)
```

## Response Behavior

Successful and blocked requests include these headers:
- `X-RateLimit-Limit`
- `X-RateLimit-Remaining`
- `X-RateLimit-Reset`

Blocked requests also include:
- status `429 Too Many Requests`
- header `Retry-After`
- JSON body:

```json
{"error":"rate limit exceeded"}
```

## Example

Runnable examples are available at:
- `examples/basic/main.go`
- `examples/custom_response/main.go`

Run them with:

```bash
go run ./examples/basic
go run ./examples/custom_response
```

## Current Scope

This library is intentionally small.

Current limitations:
- in-memory only
- single-process only
- fixed-window algorithm only
- no distributed/shared backend
