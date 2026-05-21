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

    middleware := ratelimitx.HTTPMiddleware(limiter, ratelimitx.ByIP())

    if err := http.ListenAndServe(":8080", middleware(mux)); err != nil {
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

A runnable example is available at `examples/basic/main.go`.

Run it with:

```bash
go run ./examples/basic
```

## Current Scope

This library is intentionally small.

Current limitations:
- in-memory only
- single-process only
- fixed-window algorithm only
- no distributed/shared backend
